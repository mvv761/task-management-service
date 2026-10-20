package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"task-management-service/internal/domain"
)

type TaskCommentRepository interface {
	Create(ctx context.Context, comment *domain.TaskComment) error
	FindByID(ctx context.Context, id uint64) (*domain.TaskComment, error)
	ListByTaskID(ctx context.Context, taskID uint64) ([]domain.TaskComment, error)
}

type TaskCommentTaskFinder interface {
	FindByID(ctx context.Context, id uint64) (*domain.Task, error)
}

type TaskCommentMemberFinder interface {
	FindMember(ctx context.Context, teamID, userID uint64) (*domain.TeamMember, error)
}

type TaskCommentService struct {
	commentRepository TaskCommentRepository
	taskFinder        TaskCommentTaskFinder
	memberFinder      TaskCommentMemberFinder
}

func NewTaskCommentService(
	commentRepository TaskCommentRepository,
	taskFinder TaskCommentTaskFinder,
	memberFinder TaskCommentMemberFinder,
) *TaskCommentService {
	return &TaskCommentService{
		commentRepository: commentRepository,
		taskFinder:        taskFinder,
		memberFinder:      memberFinder,
	}
}

func (s *TaskCommentService) CreateComment(ctx context.Context, userID uint64, taskID uint64, content string) (*domain.TaskComment, error) {
	task, err := s.taskFinder.FindByID(ctx, taskID)
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
		return nil, fmt.Errorf("find task member: %w", err)
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return nil, domain.ErrInvalidCommentContent
	}

	taskComment := domain.TaskComment{
		UserID:  userID,
		TaskID:  taskID,
		Content: content,
	}

	if err := s.commentRepository.Create(ctx, &taskComment); err != nil {
		return nil, fmt.Errorf("create task comment: %w", err)
	}

	comment, err := s.commentRepository.FindByID(
		ctx,
		taskComment.ID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"find created task comment: %w",
			err,
		)
	}

	return comment, nil
}

func (s *TaskCommentService) ListComments(ctx context.Context, userID uint64, taskID uint64) ([]domain.TaskComment, error) {
	task, err := s.taskFinder.FindByID(ctx, taskID)
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
		return nil, fmt.Errorf("find task member: %w", err)
	}

	comments, err := s.commentRepository.ListByTaskID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("list task comments: %w", err)
	}

	return comments, nil
}
