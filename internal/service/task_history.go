package service

import (
	"context"
	"errors"
	"fmt"

	"task-management-service/internal/domain"
)

type TaskHistoryRepository interface {
	ListByTaskID(ctx context.Context, taskID uint64) ([]domain.TaskHistory, error)
}

type TaskFinder interface {
	FindByID(ctx context.Context, id uint64) (*domain.Task, error)
}

type TaskHistoryMemberFinder interface {
	FindMember(ctx context.Context, teamID, userID uint64) (*domain.TeamMember, error)
}

type TaskHistoryService struct {
	taskHistoryRepository TaskHistoryRepository
	taskFinder            TaskFinder
	memberFinder          TaskHistoryMemberFinder
}

func NewTaskHistoryService(
	taskHistoryRepository TaskHistoryRepository,
	taskFinder TaskFinder,
	memberFinder TaskHistoryMemberFinder,
) *TaskHistoryService {
	return &TaskHistoryService{
		taskHistoryRepository: taskHistoryRepository,
		taskFinder:            taskFinder,
		memberFinder:          memberFinder,
	}
}

func (s *TaskHistoryService) ListTaskHistory(ctx context.Context, userID uint64, taskID uint64) ([]domain.TaskHistory, error) {
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

	history, err := s.taskHistoryRepository.ListByTaskID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("find task history: %w", err)
	}

	return history, nil

}
