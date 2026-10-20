package domain

import "time"

type TaskHistory struct {
	ID        uint64    `db:"id"`
	TaskID    uint64    `db:"task_id"`
	ChangedBy uint64    `db:"changed_by"`
	Changes   []byte    `db:"changes"`
	CreatedAt time.Time `db:"created_at"`
}
