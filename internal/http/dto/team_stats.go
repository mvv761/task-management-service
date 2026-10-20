package dto

type TaskStatusCountResponse struct {
	Status string `json:"status" enums:"todo,in_progress,done,cancelled" example:"done"`
	Count  uint64 `json:"count" example:"5"`
}

type TopAssigneeResponse struct {
	AssigneeID  uint64 `json:"assignee_id" example:"2"`
	ClosedCount uint64 `json:"closed_count" example:"3"`
}

type TeamStatsResponse struct {
	ByStatus            []TaskStatusCountResponse `json:"by_status"`
	TopAssignees        []TopAssigneeResponse     `json:"top_assignees"`
	AvgCloseTimeSeconds *float64                  `json:"avg_close_time_seconds" example:"3600"`
	TotalComments       uint64                    `json:"total_comments" example:"12"`
}
