package mysql

import (
	"context"
	"fmt"

	"task-management-service/internal/domain"
	"github.com/jmoiron/sqlx"
)

type TaskHistoryRepository struct {
	db *sqlx.DB
}

func NewTaskHistoryRepository(db *sqlx.DB) *TaskHistoryRepository {
	return &TaskHistoryRepository{db: db}
}

func (r *TaskHistoryRepository) ListByTaskID(ctx context.Context, taskID uint64) ([]domain.TaskHistory, error) {
	history := make([]domain.TaskHistory, 0)

	query := `SELECT id, task_id, changed_by, changes, created_at FROM task_history WHERE task_id = ? ORDER BY created_at DESC, id DESC`

	if err := r.db.SelectContext(ctx, &history, query, taskID); err != nil {
		return nil, fmt.Errorf("list task history: %w", err)
	}

	return history, nil
}
