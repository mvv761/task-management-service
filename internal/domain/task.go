package domain

import (
	"errors"
	"time"
)

type TaskStatus string

const (
	TaskStatusTodo       TaskStatus = "todo"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusDone       TaskStatus = "done"
	TaskStatusCancelled  TaskStatus = "cancelled"
)

var (
	ErrInvalidTaskTitle    = errors.New("task title is required")
	ErrInvalidAssignee     = errors.New("assignee must be a team member")
	ErrTaskNotFound        = errors.New("task not found")
	ErrInvalidTaskStatus   = errors.New("invalid task status")
	ErrInvalidTaskVersion  = errors.New("invalid task version")
	ErrEmptyTaskPatch      = errors.New("no task fields to update")
	ErrTaskVersionConflict = errors.New("task version conflict")
	ErrNoTaskChanges       = errors.New("task has no changes")
)

type Task struct {
	ID          uint64     `db:"id"`
	TeamID      uint64     `db:"team_id"`
	Title       string     `db:"title"`
	Description *string    `db:"description"`
	Status      TaskStatus `db:"status"`
	CreatedBy   uint64     `db:"created_by"`
	AssigneeID  *uint64    `db:"assignee_id"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
	ClosedAt    *time.Time `db:"closed_at"`
	Version     uint32     `db:"version"`
}
type TaskFilter struct {
	TeamID     uint64
	Status     *TaskStatus
	AssigneeID *uint64
	Limit      uint32
	Offset     uint32
}

type TaskPatch struct {
	Title *string

	Description    *string
	DescriptionSet bool

	Status *TaskStatus

	AssigneeID  *uint64
	AssigneeSet bool

	Version uint32
}

func (s TaskStatus) IsValid() bool {
	switch s {
	case TaskStatusTodo,
		TaskStatusInProgress,
		TaskStatusDone,
		TaskStatusCancelled:
		return true
	default:
		return false
	}
}

func (s TaskStatus) IsClosed() bool {
	return s == TaskStatusDone ||
		s == TaskStatusCancelled
}
