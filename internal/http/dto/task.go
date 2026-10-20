package dto

import "time"

type CreateTaskRequest struct {
	TeamID      uint64  `json:"team_id" binding:"required" example:"1"`
	Title       string  `json:"title" binding:"required" example:"Fix login rate limit"`
	Description *string `json:"description" example:"Reduce burst window for auth endpoints"`
	AssigneeID  *uint64 `json:"assignee_id" example:"2"`
}

type TaskResponse struct {
	ID          uint64     `json:"id" example:"1"`
	TeamID      uint64     `json:"team_id" example:"1"`
	Title       string     `json:"title" example:"Fix login rate limit"`
	Description *string    `json:"description" example:"Reduce burst window for auth endpoints"`
	Status      string     `json:"status" enums:"todo,in_progress,done,cancelled" example:"todo"`
	CreatedBy   uint64     `json:"created_by" example:"1"`
	AssigneeID  *uint64    `json:"assignee_id" example:"2"`
	CreatedAt   time.Time  `json:"created_at" example:"2026-09-12T10:15:00Z"`
	UpdatedAt   time.Time  `json:"updated_at" example:"2026-09-12T10:15:00Z"`
	ClosedAt    *time.Time `json:"closed_at" example:"2026-09-12T18:40:00Z"`
	Version     uint32     `json:"version" example:"1"`
}

type ListTasksResponse struct {
	Tasks []TaskResponse `json:"tasks"`
}

type UpdateTaskRequest struct {
	Title       Optional[string] `json:"title"`
	Description Optional[string] `json:"description"`
	Status      Optional[string] `json:"status"`
	AssigneeID  Optional[uint64] `json:"assignee_id"`
	Version     uint32           `json:"version"`
}

type UpdateTaskDocRequest struct {
	Title       *string `json:"title,omitempty" example:"Ship caching for task list"`
	Description *string `json:"description,omitempty" example:"Invalidate redis keys after task updates"`
	Status      *string `json:"status,omitempty" enums:"todo,in_progress,done,cancelled" example:"in_progress"`
	AssigneeID  *uint64 `json:"assignee_id,omitempty" example:"2"`
	Version     uint32  `json:"version" binding:"required" example:"1"`
}
