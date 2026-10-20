package service

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"task-management-service/internal/domain"
)

type fakeTaskRepository struct {
	createFunc func(ctx context.Context, task *domain.Task) error
	findFunc   func(ctx context.Context, id uint64) (*domain.Task, error)
	listFunc   func(ctx context.Context, filter domain.TaskFilter) ([]domain.Task, error)
	updateFunc func(ctx context.Context, currentTask *domain.Task, updatedTask *domain.Task, changedBy uint64, expectedVersion uint32) error

	createCalls int
	findCalls   int
	listCalls   int
	updateCalls int

	lastCreated         *domain.Task
	lastListFilter      domain.TaskFilter
	lastUpdatedTask     *domain.Task
	lastExpectedVersion uint32
}

func (f *fakeTaskRepository) Create(ctx context.Context, task *domain.Task) error {
	f.createCalls++
	copied := *task
	f.lastCreated = &copied
	if f.createFunc != nil {
		return f.createFunc(ctx, task)
	}
	task.ID = 1
	return nil
}

func (f *fakeTaskRepository) FindByID(ctx context.Context, id uint64) (*domain.Task, error) {
	f.findCalls++
	if f.findFunc != nil {
		return f.findFunc(ctx, id)
	}
	return nil, domain.ErrTaskNotFound
}

func (f *fakeTaskRepository) List(ctx context.Context, filter domain.TaskFilter) ([]domain.Task, error) {
	f.listCalls++
	f.lastListFilter = filter
	if f.listFunc != nil {
		return f.listFunc(ctx, filter)
	}
	return nil, nil
}

func (f *fakeTaskRepository) Update(ctx context.Context, currentTask *domain.Task, updatedTask *domain.Task, changedBy uint64, expectedVersion uint32) error {
	f.updateCalls++
	copied := *updatedTask
	f.lastUpdatedTask = &copied
	f.lastExpectedVersion = expectedVersion
	if f.updateFunc != nil {
		return f.updateFunc(ctx, currentTask, updatedTask, changedBy, expectedVersion)
	}
	return nil
}

type fakeMemberFinder struct {
	members map[uint64]*domain.TeamMember
	err     error

	calls     int
	lastTeam  uint64
	lastUser  uint64
	callUsers []uint64
}

func (f *fakeMemberFinder) FindMember(ctx context.Context, teamID, userID uint64) (*domain.TeamMember, error) {
	f.calls++
	f.lastTeam = teamID
	f.lastUser = userID
	f.callUsers = append(f.callUsers, userID)

	if f.err != nil {
		return nil, f.err
	}
	if f.members != nil {
		member, ok := f.members[userID]
		if !ok {
			return nil, domain.ErrTeamMemberNotFound
		}
		return member, nil
	}
	return &domain.TeamMember{TeamID: teamID, UserID: userID, Role: domain.TeamRoleMember}, nil
}

type fakeTaskCache struct {
	versionFunc    func(ctx context.Context, teamID uint64) (uint64, error)
	getFunc        func(ctx context.Context, filter domain.TaskFilter, version uint64) ([]domain.Task, bool, error)
	setFunc        func(ctx context.Context, filter domain.TaskFilter, version uint64, tasks []domain.Task, ttl time.Duration) error
	invalidateFunc func(ctx context.Context, teamID uint64) error

	versionCalls    int
	getCalls        int
	setCalls        int
	invalidateCalls int

	lastGetFilter domain.TaskFilter
	lastSetFilter domain.TaskFilter
	lastSetTTL    time.Duration
}

func (f *fakeTaskCache) Version(ctx context.Context, teamID uint64) (uint64, error) {
	f.versionCalls++
	if f.versionFunc != nil {
		return f.versionFunc(ctx, teamID)
	}
	return 0, nil
}

func (f *fakeTaskCache) Get(ctx context.Context, filter domain.TaskFilter, version uint64) ([]domain.Task, bool, error) {
	f.getCalls++
	f.lastGetFilter = filter
	if f.getFunc != nil {
		return f.getFunc(ctx, filter, version)
	}
	return nil, false, nil
}

func (f *fakeTaskCache) Set(ctx context.Context, filter domain.TaskFilter, version uint64, tasks []domain.Task, ttl time.Duration) error {
	f.setCalls++
	f.lastSetFilter = filter
	f.lastSetTTL = ttl
	if f.setFunc != nil {
		return f.setFunc(ctx, filter, version, tasks, ttl)
	}
	return nil
}

func (f *fakeTaskCache) Invalidate(ctx context.Context, teamID uint64) error {
	f.invalidateCalls++
	if f.invalidateFunc != nil {
		return f.invalidateFunc(ctx, teamID)
	}
	return nil
}

func TestTaskService_CreateTask_ValidatesAndNormalizesInput(t *testing.T) {
	description := "   "
	repo := &fakeTaskRepository{
		createFunc: func(ctx context.Context, task *domain.Task) error {
			task.ID = 11
			return nil
		},
		findFunc: func(ctx context.Context, id uint64) (*domain.Task, error) {
			return &domain.Task{
				ID:          id,
				TeamID:      1,
				Title:       "new task",
				Description: nil,
				Status:      domain.TaskStatusTodo,
				CreatedBy:   2,
				Version:     1,
			}, nil
		},
	}
	members := &fakeMemberFinder{}
	cache := &fakeTaskCache{}

	task, err := NewTaskService(repo, members, cache).CreateTask(context.Background(), 1, 2, "  new task  ", &description, nil)
	if err != nil {
		t.Fatalf("CreateTask returned error: %v", err)
	}

	if task.Title != "new task" {
		t.Fatalf("expected normalized title, got %q", task.Title)
	}
	if repo.lastCreated == nil || repo.lastCreated.Description != nil {
		t.Fatalf("expected blank description to be stored as nil, got %#v", repo.lastCreated)
	}
	if repo.lastCreated.Status != domain.TaskStatusTodo || repo.lastCreated.Version != 1 {
		t.Fatalf("expected default status todo and version 1, got status=%q version=%d", repo.lastCreated.Status, repo.lastCreated.Version)
	}
	if cache.invalidateCalls != 1 {
		t.Fatalf("expected cache invalidation after create, got %d", cache.invalidateCalls)
	}
}

func TestTaskService_CreateTask_InvalidTitleSkipsDependencies(t *testing.T) {
	repo := &fakeTaskRepository{}
	members := &fakeMemberFinder{}
	cache := &fakeTaskCache{}

	_, err := NewTaskService(repo, members, cache).CreateTask(context.Background(), 1, 2, "   ", nil, nil)
	if !errors.Is(err, domain.ErrInvalidTaskTitle) {
		t.Fatalf("expected ErrInvalidTaskTitle, got %v", err)
	}
	if members.calls != 0 || repo.createCalls != 0 || cache.invalidateCalls != 0 {
		t.Fatalf("expected no dependencies to be called, member=%d create=%d invalidate=%d", members.calls, repo.createCalls, cache.invalidateCalls)
	}
}

func TestTaskService_CreateTask_RejectsNonMemberAssignee(t *testing.T) {
	assigneeID := uint64(3)
	repo := &fakeTaskRepository{}
	members := &fakeMemberFinder{
		members: map[uint64]*domain.TeamMember{
			2: {TeamID: 1, UserID: 2, Role: domain.TeamRoleMember},
		},
	}
	cache := &fakeTaskCache{}

	_, err := NewTaskService(repo, members, cache).CreateTask(context.Background(), 1, 2, "task", nil, &assigneeID)
	if !errors.Is(err, domain.ErrInvalidAssignee) {
		t.Fatalf("expected ErrInvalidAssignee, got %v", err)
	}
	if repo.createCalls != 0 {
		t.Fatalf("expected task not to be created")
	}
}

func TestTaskService_ListTasks_TeamPermissions(t *testing.T) {
	repo := &fakeTaskRepository{}
	members := &fakeMemberFinder{err: domain.ErrTeamMemberNotFound}
	cache := &fakeTaskCache{}

	_, err := NewTaskService(repo, members, cache).ListTasks(context.Background(), 99, domain.TaskFilter{TeamID: 1})
	if !errors.Is(err, domain.ErrInsufficientPermissions) {
		t.Fatalf("expected ErrInsufficientPermissions, got %v", err)
	}
	if repo.listCalls != 0 || cache.versionCalls != 0 {
		t.Fatalf("expected list/cache to be skipped for non-member, list=%d cache=%d", repo.listCalls, cache.versionCalls)
	}
}

func TestTaskService_ListTasks_ReturnsCachedTasks(t *testing.T) {
	cachedTasks := []domain.Task{{ID: 7, TeamID: 1, Title: "cached"}}
	repo := &fakeTaskRepository{}
	members := &fakeMemberFinder{}
	cache := &fakeTaskCache{
		versionFunc: func(ctx context.Context, teamID uint64) (uint64, error) {
			return 5, nil
		},
		getFunc: func(ctx context.Context, filter domain.TaskFilter, version uint64) ([]domain.Task, bool, error) {
			if version != 5 {
				t.Fatalf("expected cache version 5, got %d", version)
			}
			return cachedTasks, true, nil
		},
	}

	tasks, err := NewTaskService(repo, members, cache).ListTasks(context.Background(), 2, domain.TaskFilter{TeamID: 1})
	if err != nil {
		t.Fatalf("ListTasks returned error: %v", err)
	}
	if !reflect.DeepEqual(tasks, cachedTasks) {
		t.Fatalf("expected cached tasks %#v, got %#v", cachedTasks, tasks)
	}
	if repo.listCalls != 0 || cache.setCalls != 0 {
		t.Fatalf("expected repository and cache set to be skipped, list=%d set=%d", repo.listCalls, cache.setCalls)
	}
}

func TestTaskService_ListTasks_CacheMissStoresRepositoryResult(t *testing.T) {
	repoTasks := []domain.Task{{ID: 8, TeamID: 1, Title: "db"}}
	repo := &fakeTaskRepository{
		listFunc: func(ctx context.Context, filter domain.TaskFilter) ([]domain.Task, error) {
			return repoTasks, nil
		},
	}
	members := &fakeMemberFinder{}
	cache := &fakeTaskCache{
		versionFunc: func(ctx context.Context, teamID uint64) (uint64, error) {
			return 9, nil
		},
	}

	tasks, err := NewTaskService(repo, members, cache).ListTasks(context.Background(), 2, domain.TaskFilter{TeamID: 1})
	if err != nil {
		t.Fatalf("ListTasks returned error: %v", err)
	}
	if !reflect.DeepEqual(tasks, repoTasks) {
		t.Fatalf("expected repository tasks %#v, got %#v", repoTasks, tasks)
	}
	if repo.lastListFilter.Limit != defaultTaskLimit {
		t.Fatalf("expected default limit %d, got %d", defaultTaskLimit, repo.lastListFilter.Limit)
	}
	if cache.setCalls != 1 || cache.lastSetFilter.Limit != defaultTaskLimit || cache.lastSetTTL != taskCacheTTL {
		t.Fatalf("expected cached repository result with default limit and ttl, set=%d filter=%#v ttl=%s", cache.setCalls, cache.lastSetFilter, cache.lastSetTTL)
	}
}

func TestTaskService_ListTasks_RedisVersionErrorFallsBackToRepository(t *testing.T) {
	repoTasks := []domain.Task{{ID: 9, TeamID: 1, Title: "db"}}
	repo := &fakeTaskRepository{
		listFunc: func(ctx context.Context, filter domain.TaskFilter) ([]domain.Task, error) {
			return repoTasks, nil
		},
	}
	cache := &fakeTaskCache{
		versionFunc: func(ctx context.Context, teamID uint64) (uint64, error) {
			return 0, errors.New("redis down")
		},
	}

	tasks, err := NewTaskService(repo, &fakeMemberFinder{}, cache).ListTasks(context.Background(), 2, domain.TaskFilter{TeamID: 1})
	if err != nil {
		t.Fatalf("ListTasks returned error: %v", err)
	}
	if !reflect.DeepEqual(tasks, repoTasks) {
		t.Fatalf("expected repository fallback tasks %#v, got %#v", repoTasks, tasks)
	}
	if cache.getCalls != 0 || cache.setCalls != 0 {
		t.Fatalf("expected cache get/set skipped after version error, get=%d set=%d", cache.getCalls, cache.setCalls)
	}
}

func TestTaskService_GetTask_MasksNonMemberAsTaskNotFound(t *testing.T) {
	repo := &fakeTaskRepository{
		findFunc: func(ctx context.Context, id uint64) (*domain.Task, error) {
			return &domain.Task{ID: id, TeamID: 1}, nil
		},
	}
	members := &fakeMemberFinder{err: domain.ErrTeamMemberNotFound}

	_, err := NewTaskService(repo, members, &fakeTaskCache{}).GetTask(context.Background(), 99, 10)
	if !errors.Is(err, domain.ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestTaskService_UpdateTask_MasksNonMemberAsTaskNotFound(t *testing.T) {
	title := "new title"
	repo := &fakeTaskRepository{
		findFunc: func(ctx context.Context, id uint64) (*domain.Task, error) {
			return &domain.Task{
				ID:        id,
				TeamID:    1,
				Title:     "old title",
				Status:    domain.TaskStatusTodo,
				CreatedBy: 2,
				Version:   1,
			}, nil
		},
	}
	members := &fakeMemberFinder{err: domain.ErrTeamMemberNotFound}

	_, err := NewTaskService(repo, members, &fakeTaskCache{}).UpdateTask(
		context.Background(),
		99,
		10,
		domain.TaskPatch{Title: &title, Version: 1},
	)
	if !errors.Is(err, domain.ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}
	if repo.updateCalls != 0 {
		t.Fatalf("expected repository update to be skipped")
	}
}

func TestTaskService_UpdateTask_RejectsStaleVersionBeforeRepositoryUpdate(t *testing.T) {
	title := "new title"
	repo := &fakeTaskRepository{
		findFunc: func(ctx context.Context, id uint64) (*domain.Task, error) {
			return &domain.Task{
				ID:        id,
				TeamID:    1,
				Title:     "old title",
				Status:    domain.TaskStatusTodo,
				CreatedBy: 2,
				Version:   2,
			}, nil
		},
	}

	_, err := NewTaskService(repo, &fakeMemberFinder{}, &fakeTaskCache{}).UpdateTask(
		context.Background(),
		2,
		10,
		domain.TaskPatch{Title: &title, Version: 1},
	)
	if !errors.Is(err, domain.ErrTaskVersionConflict) {
		t.Fatalf("expected ErrTaskVersionConflict, got %v", err)
	}
	if repo.updateCalls != 0 {
		t.Fatalf("expected repository update to be skipped")
	}
}

func TestTaskService_UpdateTask_PropagatesRepositoryVersionConflict(t *testing.T) {
	title := "new title"
	repo := &fakeTaskRepository{
		findFunc: func(ctx context.Context, id uint64) (*domain.Task, error) {
			return &domain.Task{
				ID:        id,
				TeamID:    1,
				Title:     "old title",
				Status:    domain.TaskStatusTodo,
				CreatedBy: 2,
				Version:   1,
			}, nil
		},
		updateFunc: func(ctx context.Context, currentTask *domain.Task, updatedTask *domain.Task, changedBy uint64, expectedVersion uint32) error {
			return domain.ErrTaskVersionConflict
		},
	}

	_, err := NewTaskService(repo, &fakeMemberFinder{}, &fakeTaskCache{}).UpdateTask(
		context.Background(),
		2,
		10,
		domain.TaskPatch{Title: &title, Version: 1},
	)
	if !errors.Is(err, domain.ErrTaskVersionConflict) {
		t.Fatalf("expected ErrTaskVersionConflict, got %v", err)
	}
	if repo.updateCalls != 1 || repo.lastExpectedVersion != 1 {
		t.Fatalf("expected repository update with expected version 1, calls=%d version=%d", repo.updateCalls, repo.lastExpectedVersion)
	}
}

func TestTaskService_UpdateTask_Validation(t *testing.T) {
	currentTask := domain.Task{
		ID:        10,
		TeamID:    1,
		Title:     "old title",
		Status:    domain.TaskStatusTodo,
		CreatedBy: 2,
		Version:   1,
	}
	blankTitle := "   "
	badStatus := domain.TaskStatus("bad")
	newTitle := "new title"

	tests := []struct {
		name    string
		patch   domain.TaskPatch
		wantErr error
	}{
		{
			name:    "missing version",
			patch:   domain.TaskPatch{Title: &newTitle},
			wantErr: domain.ErrInvalidTaskVersion,
		},
		{
			name:    "empty patch",
			patch:   domain.TaskPatch{Version: 1},
			wantErr: domain.ErrEmptyTaskPatch,
		},
		{
			name:    "blank title",
			patch:   domain.TaskPatch{Title: &blankTitle, Version: 1},
			wantErr: domain.ErrInvalidTaskTitle,
		},
		{
			name:    "invalid status",
			patch:   domain.TaskPatch{Status: &badStatus, Version: 1},
			wantErr: domain.ErrInvalidTaskStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeTaskRepository{
				findFunc: func(ctx context.Context, id uint64) (*domain.Task, error) {
					task := currentTask
					return &task, nil
				},
			}

			_, err := NewTaskService(repo, &fakeMemberFinder{}, &fakeTaskCache{}).UpdateTask(context.Background(), 2, currentTask.ID, tt.patch)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			if repo.updateCalls != 0 {
				t.Fatalf("expected repository update to be skipped")
			}
		})
	}
}

func TestTaskService_UpdateTask_StatusClosesAndReopensTask(t *testing.T) {
	done := domain.TaskStatusDone
	repo := &fakeTaskRepository{
		findFunc: func(ctx context.Context, id uint64) (*domain.Task, error) {
			return &domain.Task{
				ID:        id,
				TeamID:    1,
				Title:     "task",
				Status:    domain.TaskStatusTodo,
				CreatedBy: 2,
				Version:   1,
			}, nil
		},
	}
	cache := &fakeTaskCache{}

	_, err := NewTaskService(repo, &fakeMemberFinder{}, cache).UpdateTask(context.Background(), 2, 10, domain.TaskPatch{Status: &done, Version: 1})
	if err != nil {
		t.Fatalf("UpdateTask returned error: %v", err)
	}
	if repo.lastUpdatedTask == nil || repo.lastUpdatedTask.ClosedAt == nil {
		t.Fatalf("expected ClosedAt to be set when task is closed")
	}
	if cache.invalidateCalls != 1 {
		t.Fatalf("expected cache invalidation after update, got %d", cache.invalidateCalls)
	}
}
