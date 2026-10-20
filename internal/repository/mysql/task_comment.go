package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"task-management-service/internal/domain"
	"github.com/jmoiron/sqlx"
)

type TaskCommentRepository struct {
	db *sqlx.DB
}

func NewTaskCommentRepository(
	db *sqlx.DB,
) *TaskCommentRepository {
	return &TaskCommentRepository{
		db: db,
	}
}

func (r *TaskCommentRepository) Create(ctx context.Context, comment *domain.TaskComment) error {
	query := `INSERT INTO task_comments (task_id, user_id, content) VALUES (?, ?, ?)`

	res, err := r.db.ExecContext(ctx, query, comment.TaskID, comment.UserID, comment.Content)
	if err != nil {
		return fmt.Errorf("create task comment: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("get task comment id: %w", err)
	}

	comment.ID = uint64(id)

	return nil
}

func (r *TaskCommentRepository) ListByTaskID(ctx context.Context, taskID uint64) ([]domain.TaskComment, error) {
	comments := make([]domain.TaskComment, 0)

	query := `SELECT id, task_id, user_id, content, created_at FROM task_comments WHERE task_id = ? ORDER BY created_at ASC, id ASC`

	err := r.db.SelectContext(ctx, &comments, query, taskID)
	if err != nil {
		return nil, fmt.Errorf("list task comments: %w", err)
	}

	return comments, nil
}

func (r *TaskCommentRepository) FindByID(ctx context.Context, id uint64) (*domain.TaskComment, error) {
	var taskComment domain.TaskComment
	query := `SELECT id, task_id, user_id, content, created_at FROM task_comments WHERE id = ?`

	err := r.db.GetContext(ctx, &taskComment, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrTaskCommentNotFound
		}
		return nil, fmt.Errorf("find task_comment by id: %w", err)
	}

	return &taskComment, nil
}
