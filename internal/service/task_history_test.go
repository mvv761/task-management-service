package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"task-management-service/internal/domain"
)

type fakeTaskHistoryRepository struct {
	listFunc func(ctx context.Context, taskID uint64) ([]domain.TaskHistory, error)
	calls    int
}

func (f *fakeTaskHistoryRepository) ListByTaskID(ctx context.Context, taskID uint64) ([]domain.TaskHistory, error) {
	f.calls++
	if f.listFunc != nil {
		return f.listFunc(ctx, taskID)
	}
	return nil, nil
}

func TestTaskHistoryService_ListTaskHistory_ReturnsHistoryForTeamMember(t *testing.T) {
	want := []domain.TaskHistory{
		{ID: 1, TaskID: 10, ChangedBy: 2, Changes: []byte(`{"status":{"old":"todo","new":"done"}}`)},
	}
	historyRepo := &fakeTaskHistoryRepository{
		listFunc: func(ctx context.Context, taskID uint64) ([]domain.TaskHistory, error) {
			return want, nil
		},
	}
	taskRepo := &fakeTaskRepository{
		findFunc: func(ctx context.Context, id uint64) (*domain.Task, error) {
			return &domain.Task{ID: id, TeamID: 1}, nil
		},
	}

	got, err := NewTaskHistoryService(historyRepo, taskRepo, &fakeMemberFinder{}).ListTaskHistory(context.Background(), 2, 10)
	if err != nil {
		t.Fatalf("ListTaskHistory returned error: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

func TestTaskHistoryService_ListTaskHistory_MasksNonMemberAsTaskNotFound(t *testing.T) {
	historyRepo := &fakeTaskHistoryRepository{}
	taskRepo := &fakeTaskRepository{
		findFunc: func(ctx context.Context, id uint64) (*domain.Task, error) {
			return &domain.Task{ID: id, TeamID: 1}, nil
		},
	}

	_, err := NewTaskHistoryService(historyRepo, taskRepo, &fakeMemberFinder{err: domain.ErrTeamMemberNotFound}).ListTaskHistory(context.Background(), 99, 10)
	if !errors.Is(err, domain.ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}
	if historyRepo.calls != 0 {
		t.Fatalf("expected history repository to be skipped")
	}
}
