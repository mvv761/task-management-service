package dto

import (
	"encoding/json"
	"time"
)

type TaskHistoryResponse struct {
	ID        uint64          `json:"id" example:"1"`
	TaskID    uint64          `json:"task_id" example:"1"`
	ChangedBy uint64          `json:"changed_by" example:"1"`
	Changes   json.RawMessage `json:"changes" swaggertype:"object"`
	CreatedAt time.Time       `json:"created_at" example:"2026-08-25T12:00:00Z"`
}

type ListTaskHistoryResponse struct {
	History []TaskHistoryResponse `json:"history"`
}
