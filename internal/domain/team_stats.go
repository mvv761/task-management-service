package domain

type TaskStatusCount struct {
	Status TaskStatus
	Count  uint64
}

type TopAssignee struct {
	AssigneeID  uint64
	ClosedCount uint64
}

type TeamStats struct {
	ByStatus            []TaskStatusCount
	TopAssignees        []TopAssignee
	AvgCloseTimeSeconds *float64
	TotalComments       uint64
}
