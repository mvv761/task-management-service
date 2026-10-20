package domain

import (
	"errors"
	"time"
)

var (
	ErrInvalidCommentContent = errors.New("comment content is required")
	ErrTaskCommentNotFound   = errors.New("task comment not found")
)

type TaskComment struct {
	ID        uint64    `db:"id"`
	TaskID    uint64    `db:"task_id"`
	UserID    uint64    `db:"user_id"`
	Content   string    `db:"content"`
	CreatedAt time.Time `db:"created_at"`
}
