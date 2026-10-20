package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"task-management-service/internal/domain"
	"github.com/jmoiron/sqlx"
)

type TaskRepository struct {
	db *sqlx.DB
}

func NewTaskRepository(db *sqlx.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

type taskFieldChange struct {
	Old any `json:"old"`
	New any `json:"new"`
}

func equalPtr[T comparable](a, b *T) bool {
	if a == nil || b == nil {
		return a == b
	}

	return *a == *b
}

func equalTimePtr(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == b
	}

	return a.Equal(*b)
}

func buildTaskChanges(
	currentTask *domain.Task,
	updatedTask *domain.Task,
) map[string]taskFieldChange {
	changes := make(map[string]taskFieldChange)

	if currentTask.Title != updatedTask.Title {
		changes["title"] = taskFieldChange{
			Old: currentTask.Title,
			New: updatedTask.Title,
		}
	}

	if !equalPtr(
		currentTask.Description,
		updatedTask.Description,
	) {
		changes["description"] = taskFieldChange{
			Old: currentTask.Description,
			New: updatedTask.Description,
		}
	}

	if currentTask.Status != updatedTask.Status {
		changes["status"] = taskFieldChange{
			Old: currentTask.Status,
			New: updatedTask.Status,
		}
	}

	if !equalPtr(
		currentTask.AssigneeID,
		updatedTask.AssigneeID,
	) {
		changes["assignee_id"] = taskFieldChange{
			Old: currentTask.AssigneeID,
			New: updatedTask.AssigneeID,
		}
	}

	if !equalTimePtr(
		currentTask.ClosedAt,
		updatedTask.ClosedAt,
	) {
		changes["closed_at"] = taskFieldChange{
			Old: currentTask.ClosedAt,
			New: updatedTask.ClosedAt,
		}
	}

	return changes
}

func (r *TaskRepository) Create(ctx context.Context, task *domain.Task) error {
	query := `INSERT INTO tasks (team_id, title, description, status, created_by, assignee_id) VALUES (?, ?, ?, ?, ?, ?)`

	res, err := r.db.ExecContext(ctx, query, task.TeamID, task.Title, task.Description, task.Status, task.CreatedBy, task.AssigneeID)
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("get task id: %w", err)
	}

	task.ID = uint64(id)

	return nil
}

func (r *TaskRepository) FindByID(ctx context.Context, id uint64) (*domain.Task, error) {
	var task domain.Task
	query := `SELECT id, team_id, title, description, status, created_by, assignee_id, created_at, updated_at, closed_at, version FROM tasks WHERE id = ?`
	err := r.db.GetContext(ctx, &task, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrTaskNotFound
		}
		return nil, fmt.Errorf("find task by id: %w", err)
	}

	return &task, nil
}

func (r *TaskRepository) List(ctx context.Context, filter domain.TaskFilter) ([]domain.Task, error) {
	tasks := make([]domain.Task, 0)
	query := `SELECT id, team_id, title, description, status, created_by, assignee_id, created_at, updated_at, closed_at, version FROM tasks WHERE team_id = ?`

	args := []any{filter.TeamID}

	if filter.Status != nil {
		query += " AND status = ?"
		args = append(args, *filter.Status)
	}

	if filter.AssigneeID != nil {
		query += " AND assignee_id = ?"
		args = append(args, *filter.AssigneeID)
	}

	query += " ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?"
	args = append(args, filter.Limit, filter.Offset)

	err := r.db.SelectContext(
		ctx,
		&tasks,
		query,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}

	return tasks, nil
}

func (r *TaskRepository) Update(ctx context.Context, currentTask *domain.Task, updatedTask *domain.Task, changedBy uint64, expectedVersion uint32) error {
	changes := buildTaskChanges(currentTask, updatedTask)
	if len(changes) == 0 {
		return domain.ErrNoTaskChanges
	}

	changesJSON, err := json.Marshal(changes)
	if err != nil {
		return fmt.Errorf("marshal task changes: %w", err)
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Increment version atomically in the database.
	// Together with `WHERE id = ? AND version = ?` this implements
	// optimistic locking and prevents lost updates between concurrent requests.
	query := `UPDATE tasks SET title = ?, description = ?, status = ?, assignee_id = ?, closed_at = ?, updated_at = NOW(), version = version + 1 WHERE id = ? AND version = ?`

	res, err := tx.ExecContext(
		ctx,
		query,
		updatedTask.Title,
		updatedTask.Description,
		updatedTask.Status,
		updatedTask.AssigneeID,
		updatedTask.ClosedAt,
		updatedTask.ID,
		expectedVersion,
	)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("get updated task rows affected: %w", err)
	}

	if affected == 0 {
		return domain.ErrTaskVersionConflict
	}

	historyQuery := `
	INSERT INTO task_history (
		task_id,
		changed_by,
		changes
	)
	VALUES (?, ?, ?)
`

	_, err = tx.ExecContext(
		ctx,
		historyQuery,
		updatedTask.ID,
		changedBy,
		string(changesJSON),
	)
	if err != nil {
		return fmt.Errorf("create task history: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit task update transaction: %w", err)
	}

	return nil
}
