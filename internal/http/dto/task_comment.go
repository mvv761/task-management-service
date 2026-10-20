package dto

import "time"

type CreateTaskCommentRequest struct {
	Content string `json:"content" binding:"required" example:"Looks good, ship it"`
}

type TaskCommentResponse struct {
	ID        uint64    `json:"id" example:"1"`
	TaskID    uint64    `json:"task_id" example:"1"`
	UserID    uint64    `json:"user_id" example:"1"`
	Content   string    `json:"content" example:"Looks good, ship it"`
	CreatedAt time.Time `json:"created_at" example:"2026-09-12T11:00:00Z"`
}

type ListTaskCommentsResponse struct {
	Comments []TaskCommentResponse `json:"comments"`
}
