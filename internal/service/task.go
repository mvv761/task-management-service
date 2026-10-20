package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"task-management-service/internal/domain"
	"task-management-service/internal/policy"
)

const (
	defaultTaskLimit uint32 = 20
	maxTaskLimit     uint32 = 100
	taskCacheTTL            = 5 * time.Minute
)

type TaskRepository interface {
	Create(ctx context.Context, task *domain.Task) error
	FindByID(ctx context.Context, id uint64) (*domain.Task, error)
	List(ctx context.Context, filter domain.TaskFilter) ([]domain.Task, error)
	Update(
		ctx context.Context,
		currentTask *domain.Task,
		updatedTask *domain.Task,
		changedBy uint64,
		expectedVersion uint32,
	) error
}

type TaskMemberFinder interface {
	FindMember(ctx context.Context, teamID, userID uint64) (*domain.TeamMember, error)
}

type TaskCache interface {
	Version(
		ctx context.Context,
		teamID uint64,
	) (uint64, error)

	Get(
		ctx context.Context,
		filter domain.TaskFilter,
		version uint64,
	) ([]domain.Task, bool, error)

	Set(
		ctx context.Context,
		filter domain.TaskFilter,
		version uint64,
		tasks []domain.Task,
		ttl time.Duration,
	) error

	Invalidate(
		ctx context.Context,
		teamID uint64,
	) error
}

type TaskService struct {
	taskRepository TaskRepository
	memberFinder   TaskMemberFinder
	cache          TaskCache
	logger         *slog.Logger
}

func NewTaskService(taskRepository TaskRepository, memberFinder TaskMemberFinder, cache TaskCache, logger ...*slog.Logger) *TaskService {
	taskLogger := slog.Default()
	if len(logger) > 0 && logger[0] != nil {
		taskLogger = logger[0]
	}

	return &TaskService{
		taskRepository: taskRepository,
		memberFinder:   memberFinder,
		cache:          cache,
		logger:         taskLogger,
	}
}

func (s *TaskService) CreateTask(ctx context.Context, teamID uint64, userID uint64, title string, description *string, assigneeID *uint64) (*domain.Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, domain.ErrInvalidTaskTitle
	}

	if description != nil {
		trimmed := strings.TrimSpace(*description)

		if trimmed == "" {
			description = nil
		} else {
			description = &trimmed
		}
	}

	_, err := s.memberFinder.FindMember(ctx, teamID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrTeamMemberNotFound) {
			return nil, domain.ErrInsufficientPermissions
		}
		return nil, fmt.Errorf("check task creator membership: %w", err)
	}

	if assigneeID != nil {
		_, err := s.memberFinder.FindMember(ctx, teamID, *assigneeID)
		if err != nil {
			if errors.Is(err, domain.ErrTeamMemberNotFound) {
				return nil, domain.ErrInvalidAssignee
			}
			return nil, fmt.Errorf("check assignee membership: %w", err)
		}
	}

	task := domain.Task{
		TeamID:      teamID,
		Title:       title,
		Description: description,
		Status:      domain.TaskStatusTodo,
		CreatedBy:   userID,
		AssigneeID:  assigneeID,
		Version:     1,
	}

	if err := s.taskRepository.Create(ctx, &task); err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	if err := s.cache.Invalidate(ctx, task.TeamID); err != nil {
		s.logger.Warn(
			"task cache invalidate failed",
			"error", err,
			"team_id", task.TeamID,
		)
	}

	createdTask, err := s.taskRepository.FindByID(ctx, task.ID)
	if err != nil {
		return nil, fmt.Errorf("find created task: %w", err)
	}

	return createdTask, nil
}

func (s *TaskService) ListTasks(ctx context.Context, userID uint64, filter domain.TaskFilter) ([]domain.Task, error) {
	_, err := s.memberFinder.FindMember(ctx, filter.TeamID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrTeamMemberNotFound) {
			return nil, domain.ErrInsufficientPermissions
		}
		return nil, fmt.Errorf("check task list membership: %w", err)
	}

	if filter.Status != nil && !filter.Status.IsValid() {
		return nil, domain.ErrInvalidTaskStatus
	}

	if filter.Limit == 0 {
		filter.Limit = defaultTaskLimit
	}

	if filter.Limit > maxTaskLimit {
		filter.Limit = maxTaskLimit
	}

	version, versionErr := s.cache.Version(
		ctx,
		filter.TeamID,
	)

	if versionErr == nil {
		cachedTasks, found, cacheErr := s.cache.Get(
			ctx,
			filter,
			version,
		)

		if cacheErr == nil && found {
			return cachedTasks, nil
		}

		if cacheErr != nil {
			s.logger.Warn(
				"task cache get failed",
				"error", cacheErr,
				"team_id", filter.TeamID,
			)
		}
	} else {
		s.logger.Warn(
			"task cache version failed",
			"error", versionErr,
			"team_id", filter.TeamID,
		)
	}

	tasks, err := s.taskRepository.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	if versionErr == nil {
		if err := s.cache.Set(
			ctx,
			filter,
			version,
			tasks,
			taskCacheTTL,
		); err != nil {
			s.logger.Warn(
				"task cache set failed",
				"error", err,
				"team_id", filter.TeamID,
			)
		}
	}
	return tasks, nil
}

func (s *TaskService) GetTask(ctx context.Context, userID uint64, taskID uint64) (*domain.Task, error) {
	task, err := s.taskRepository.FindByID(ctx, taskID)
	if err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			return nil, domain.ErrTaskNotFound
		}
		return nil, fmt.Errorf("find task: %w", err)
	}

	_, err = s.memberFinder.FindMember(ctx, task.TeamID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrTeamMemberNotFound) {
			return nil, domain.ErrTaskNotFound
		}
		return nil, fmt.Errorf("check task membership: %w", err)
	}

	return task, nil
}

func (s *TaskService) UpdateTask(ctx context.Context, userID uint64, taskID uint64, patch domain.TaskPatch) (*domain.Task, error) {
	if patch.Version == 0 {
		return nil, domain.ErrInvalidTaskVersion
	}

	if patch.Title == nil &&
		!patch.DescriptionSet &&
		patch.Status == nil &&
		!patch.AssigneeSet {
		return nil, domain.ErrEmptyTaskPatch
	}

	task, err := s.taskRepository.FindByID(ctx, taskID)
	if err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			return nil, domain.ErrTaskNotFound
		}

		return nil, fmt.Errorf("find task: %w", err)
	}

	actor, err := s.memberFinder.FindMember(
		ctx,
		task.TeamID,
		userID,
	)
	if err != nil {
		if errors.Is(err, domain.ErrTeamMemberNotFound) {
			return nil, domain.ErrTaskNotFound
		}

		return nil, fmt.Errorf("find task actor: %w", err)
	}

	if err := policy.CanEditTask(*actor, *task, patch); err != nil {
		return nil, err
	}

	if task.Version != patch.Version {
		return nil, domain.ErrTaskVersionConflict
	}

	if patch.Title != nil {
		title := strings.TrimSpace(*patch.Title)
		if title == "" {
			return nil, domain.ErrInvalidTaskTitle
		}

		patch.Title = &title
	}

	if patch.DescriptionSet && patch.Description != nil {
		description := strings.TrimSpace(*patch.Description)
		if description == "" {
			patch.Description = nil
		} else {
			patch.Description = &description
		}
	}

	if patch.Status != nil && !patch.Status.IsValid() {
		return nil, domain.ErrInvalidTaskStatus
	}

	if patch.AssigneeSet && patch.AssigneeID != nil {
		_, err := s.memberFinder.FindMember(ctx, task.TeamID, *patch.AssigneeID)
		if err != nil {
			if errors.Is(err, domain.ErrTeamMemberNotFound) {
				return nil, domain.ErrInvalidAssignee
			}

			return nil, fmt.Errorf("check assignee membership: %w", err)
		}
	}

	updatedTask := *task

	if patch.Title != nil {
		updatedTask.Title = *patch.Title
	}

	if patch.DescriptionSet {
		updatedTask.Description = patch.Description
	}

	if patch.AssigneeSet {
		updatedTask.AssigneeID = patch.AssigneeID
	}

	if patch.Status != nil {
		wasClosed := task.Status.IsClosed()

		updatedTask.Status = *patch.Status
		isClosed := updatedTask.Status.IsClosed()

		switch {
		case !wasClosed && isClosed:
			now := time.Now().UTC()
			updatedTask.ClosedAt = &now

		case wasClosed && !isClosed:
			updatedTask.ClosedAt = nil
		}
	}

	if err := s.taskRepository.Update(ctx, task, &updatedTask, userID, patch.Version); err != nil {
		switch {
		case errors.Is(err, domain.ErrTaskVersionConflict):
			return nil, domain.ErrTaskVersionConflict

		case errors.Is(err, domain.ErrNoTaskChanges):
			return nil, domain.ErrNoTaskChanges

		default:
			return nil, fmt.Errorf("update task: %w", err)
		}
	}

	if err := s.cache.Invalidate(ctx, task.TeamID); err != nil {
		s.logger.Warn(
			"task cache invalidate failed",
			"error", err,
			"team_id", task.TeamID,
		)
	}

	result, err := s.taskRepository.FindByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("find updated task: %w", err)
	}

	return result, nil
}
