package service

import (
	"context"
	"errors"
	"testing"

	"task-management-service/internal/domain"
)

type fakeTaskCommentRepository struct {
	createCalls int
	listCalls   int
}

func (f *fakeTaskCommentRepository) Create(ctx context.Context, comment *domain.TaskComment) error {
	f.createCalls++
	comment.ID = 1
	return nil
}

func (f *fakeTaskCommentRepository) FindByID(ctx context.Context, id uint64) (*domain.TaskComment, error) {
	return &domain.TaskComment{ID: id, TaskID: 10, UserID: 2, Content: "comment"}, nil
}

func (f *fakeTaskCommentRepository) ListByTaskID(ctx context.Context, taskID uint64) ([]domain.TaskComment, error) {
	f.listCalls++
	return []domain.TaskComment{{ID: 1, TaskID: taskID, UserID: 2, Content: "comment"}}, nil
}

func TestTaskCommentService_CreateComment_MasksNonMemberAsTaskNotFound(t *testing.T) {
	commentRepo := &fakeTaskCommentRepository{}
	taskRepo := &fakeTaskRepository{
		findFunc: func(ctx context.Context, id uint64) (*domain.Task, error) {
			return &domain.Task{ID: id, TeamID: 1}, nil
		},
	}
	members := &fakeMemberFinder{err: domain.ErrTeamMemberNotFound}

	_, err := NewTaskCommentService(commentRepo, taskRepo, members).CreateComment(context.Background(), 99, 10, "comment")
	if !errors.Is(err, domain.ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}
	if commentRepo.createCalls != 0 {
		t.Fatalf("expected comment repository create to be skipped")
	}
}

func TestTaskCommentService_ListComments_MasksNonMemberAsTaskNotFound(t *testing.T) {
	commentRepo := &fakeTaskCommentRepository{}
	taskRepo := &fakeTaskRepository{
		findFunc: func(ctx context.Context, id uint64) (*domain.Task, error) {
			return &domain.Task{ID: id, TeamID: 1}, nil
		},
	}
	members := &fakeMemberFinder{err: domain.ErrTeamMemberNotFound}

	_, err := NewTaskCommentService(commentRepo, taskRepo, members).ListComments(context.Background(), 99, 10)
	if !errors.Is(err, domain.ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}
	if commentRepo.listCalls != 0 {
		t.Fatalf("expected comment repository list to be skipped")
	}
}
