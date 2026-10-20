package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"task-management-service/internal/domain"
	"github.com/jmoiron/sqlx"
)

type teamStatsRow struct {
	ByStatus            []byte          `db:"by_status"`
	TopAssignees        []byte          `db:"top_assignees"`
	AvgCloseTimeSeconds sql.NullFloat64 `db:"avg_close_time_seconds"`
	TotalComments       uint64          `db:"total_comments"`
}

type taskStatusCountJSON struct {
	Status domain.TaskStatus `json:"status"`
	Count  uint64            `json:"count"`
}

type topAssigneeJSON struct {
	AssigneeID  uint64 `json:"assignee_id"`
	ClosedCount uint64 `json:"closed_count"`
}

type TeamStatsRepository struct {
	db *sqlx.DB
}

func NewTeamStatsRepository(
	db *sqlx.DB,
) *TeamStatsRepository {
	return &TeamStatsRepository{
		db: db,
	}
}

func (r *TeamStatsRepository) GetByTeamID(ctx context.Context, teamID uint64) (*domain.TeamStats, error) {
	query := `
		WITH team_tasks AS (
			SELECT
				id,
				status,
				assignee_id,
				created_at,
				closed_at
			FROM tasks
			WHERE team_id = ?
		),
		status_counts AS (
			SELECT
				status,
				COUNT(*) AS cnt
			FROM team_tasks
			GROUP BY status
		),
		top_assignees AS (
    		SELECT
    		    assignee_id,
    		    COUNT(*) AS closed_cnt
    		FROM team_tasks
    		WHERE status = 'done'
    		  AND closed_at >= NOW() - INTERVAL 30 DAY
    		  AND assignee_id IS NOT NULL
    		GROUP BY assignee_id
    		ORDER BY closed_cnt DESC, assignee_id ASC
    		LIMIT 3
		),
		avg_close AS (
			SELECT 
				AVG(
					TIMESTAMPDIFF(SECOND, created_at, closed_at)
				) AS avg_close_time_seconds
			FROM team_tasks
			WHERE closed_at IS NOT NULL
		),
		comments_total AS (
		    SELECT
		        COUNT(*) AS cnt
		    FROM task_comments tc
		    JOIN team_tasks t
		        ON t.id = tc.task_id
		)
		SELECT
			COALESCE(
				(
					SELECT JSON_ARRAYAGG(
						JSON_OBJECT(
							'status', status,
							'count', cnt
						)
					)
					FROM status_counts
				),
				JSON_ARRAY()
			) AS by_status,
		
			COALESCE(
				(
					SELECT JSON_ARRAYAGG(
						JSON_OBJECT(
							'assignee_id', assignee_id,
							'closed_count', closed_cnt
						)
					)
					FROM top_assignees
				),
				JSON_ARRAY()
			) AS top_assignees,
		
			(
				SELECT avg_close_time_seconds
				FROM avg_close
			) AS avg_close_time_seconds,
		
			(
				SELECT cnt
				FROM comments_total
			) AS total_comments;
			`

	var row teamStatsRow
	err := r.db.GetContext(ctx, &row, query, teamID)
	if err != nil {
		return nil, fmt.Errorf("get team stats: %w", err)
	}

	var statusRows []taskStatusCountJSON

	if err := json.Unmarshal(
		row.ByStatus,
		&statusRows,
	); err != nil {
		return nil, fmt.Errorf(
			"unmarshal task status stats: %w",
			err,
		)
	}

	var topAssigneeRows []topAssigneeJSON

	if err := json.Unmarshal(
		row.TopAssignees,
		&topAssigneeRows,
	); err != nil {
		return nil, fmt.Errorf(
			"unmarshal top assignees: %w",
			err,
		)
	}

	stats := &domain.TeamStats{
		ByStatus:      make([]domain.TaskStatusCount, 0, len(statusRows)),
		TopAssignees:  make([]domain.TopAssignee, 0, len(topAssigneeRows)),
		TotalComments: row.TotalComments,
	}

	for _, statusRow := range statusRows {
		stats.ByStatus = append(stats.ByStatus, domain.TaskStatusCount{
			Status: statusRow.Status,
			Count:  statusRow.Count,
		})
	}

	for _, assigneeRow := range topAssigneeRows {
		stats.TopAssignees = append(stats.TopAssignees, domain.TopAssignee{
			AssigneeID:  assigneeRow.AssigneeID,
			ClosedCount: assigneeRow.ClosedCount,
		})
	}

	if row.AvgCloseTimeSeconds.Valid {
		avg := row.AvgCloseTimeSeconds.Float64
		stats.AvgCloseTimeSeconds = &avg
	}

	return stats, nil

}
