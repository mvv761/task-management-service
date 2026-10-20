package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"task-management-service/internal/domain"
	_ "github.com/go-sql-driver/mysql"
	migrate "github.com/golang-migrate/migrate/v4"
	migratemysql "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
)

func TestTeamStatsRepository_GetByTeamID_Integration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	db := startTeamStatsMySQL(t, ctx)

	ownerEmail := "stats-owner@example.test"
	assigneeOneEmail := "stats-assignee-one@example.test"
	assigneeTwoEmail := "stats-assignee-two@example.test"
	assigneeThreeEmail := "stats-assignee-three@example.test"
	assigneeFourEmail := "stats-assignee-four@example.test"
	commenterEmail := "stats-commenter@example.test"

	ownerID := insertTeamStatsUser(t, ctx, db, ownerEmail)
	assigneeOneID := insertTeamStatsUser(t, ctx, db, assigneeOneEmail)
	assigneeTwoID := insertTeamStatsUser(t, ctx, db, assigneeTwoEmail)
	assigneeThreeID := insertTeamStatsUser(t, ctx, db, assigneeThreeEmail)
	assigneeFourID := insertTeamStatsUser(t, ctx, db, assigneeFourEmail)
	commenterID := insertTeamStatsUser(t, ctx, db, commenterEmail)
	teamID := insertTeamStatsTeam(t, ctx, db, ownerID)
	anotherTeamID := insertTeamStatsTeam(t, ctx, db, ownerID)

	insertTeamStatsTask(t, ctx, db, teamID, "todo task", domain.TaskStatusTodo, ownerID, assigneeOneID, 0, 0)
	insertTeamStatsTask(t, ctx, db, teamID, "in progress task", domain.TaskStatusInProgress, ownerID, assigneeOneID, 0, 0)
	doneOneID := insertTeamStatsTask(t, ctx, db, teamID, "done one", domain.TaskStatusDone, ownerID, assigneeOneID, 3, 1)
	doneTwoID := insertTeamStatsTask(t, ctx, db, teamID, "done two", domain.TaskStatusDone, ownerID, assigneeOneID, 5, 2)
	doneThreeID := insertTeamStatsTask(t, ctx, db, teamID, "done three", domain.TaskStatusDone, ownerID, assigneeTwoID, 6, 3)
	oldDoneID := insertTeamStatsTask(t, ctx, db, teamID, "old done", domain.TaskStatusDone, ownerID, assigneeTwoID, 800, 799)
	insertTeamStatsTask(t, ctx, db, teamID, "done assignee one extra one", domain.TaskStatusDone, ownerID, assigneeOneID, 8, 7)
	insertTeamStatsTask(t, ctx, db, teamID, "done assignee one extra two", domain.TaskStatusDone, ownerID, assigneeOneID, 10, 9)
	insertTeamStatsTask(t, ctx, db, teamID, "done assignee two extra one", domain.TaskStatusDone, ownerID, assigneeTwoID, 12, 11)
	insertTeamStatsTask(t, ctx, db, teamID, "done assignee two extra two", domain.TaskStatusDone, ownerID, assigneeTwoID, 14, 13)
	insertTeamStatsTask(t, ctx, db, teamID, "done assignee three one", domain.TaskStatusDone, ownerID, assigneeThreeID, 16, 15)
	insertTeamStatsTask(t, ctx, db, teamID, "done assignee three two", domain.TaskStatusDone, ownerID, assigneeThreeID, 18, 17)
	insertTeamStatsTask(t, ctx, db, teamID, "done assignee four one", domain.TaskStatusDone, ownerID, assigneeFourID, 20, 19)

	insertTeamStatsComment(t, ctx, db, doneOneID, commenterID)
	insertTeamStatsComment(t, ctx, db, doneTwoID, commenterID)
	insertTeamStatsComment(t, ctx, db, doneThreeID, commenterID)
	insertTeamStatsComment(t, ctx, db, oldDoneID, commenterID)

	anotherDoneOneID := insertTeamStatsTask(t, ctx, db, anotherTeamID, "another done one", domain.TaskStatusDone, ownerID, assigneeOneID, 4, 1)
	anotherDoneTwoID := insertTeamStatsTask(t, ctx, db, anotherTeamID, "another done two", domain.TaskStatusDone, ownerID, assigneeTwoID, 5, 2)
	anotherDoneThreeID := insertTeamStatsTask(t, ctx, db, anotherTeamID, "another done three", domain.TaskStatusDone, ownerID, assigneeTwoID, 6, 3)

	insertTeamStatsComment(t, ctx, db, anotherDoneOneID, commenterID)
	insertTeamStatsComment(t, ctx, db, anotherDoneOneID, commenterID)
	insertTeamStatsComment(t, ctx, db, anotherDoneTwoID, commenterID)
	insertTeamStatsComment(t, ctx, db, anotherDoneTwoID, commenterID)
	insertTeamStatsComment(t, ctx, db, anotherDoneThreeID, commenterID)

	stats, err := NewTeamStatsRepository(db).GetByTeamID(ctx, teamID)
	if err != nil {
		t.Fatalf("GetByTeamID returned error: %v", err)
	}

	statusCounts := make(map[domain.TaskStatus]uint64, len(stats.ByStatus))
	for _, item := range stats.ByStatus {
		statusCounts[item.Status] = item.Count
	}

	wantStatusCounts := map[domain.TaskStatus]uint64{
		domain.TaskStatusTodo:       1,
		domain.TaskStatusInProgress: 1,
		domain.TaskStatusDone:       11,
	}
	if len(statusCounts) != len(wantStatusCounts) {
		t.Fatalf("expected status counts %#v, got %#v", wantStatusCounts, statusCounts)
	}
	for status, want := range wantStatusCounts {
		if statusCounts[status] != want {
			t.Fatalf("expected status %q count %d, got %d in %#v", status, want, statusCounts[status], statusCounts)
		}
	}

	wantTopAssignees := map[uint64]uint64{
		assigneeOneID:   4,
		assigneeTwoID:   3,
		assigneeThreeID: 2,
	}
	topAssignees := make(map[uint64]uint64, len(stats.TopAssignees))
	for _, assignee := range stats.TopAssignees {
		if _, exists := topAssignees[assignee.AssigneeID]; exists {
			t.Fatalf("duplicate top assignee %d in %#v", assignee.AssigneeID, stats.TopAssignees)
		}
		topAssignees[assignee.AssigneeID] = assignee.ClosedCount
	}
	if len(topAssignees) != len(wantTopAssignees) {
		t.Fatalf("expected top assignees %#v, got %#v", wantTopAssignees, topAssignees)
	}
	for assigneeID, want := range wantTopAssignees {
		if topAssignees[assigneeID] != want {
			t.Fatalf("expected top assignee %d closed count %d, got %d in %#v", assigneeID, want, topAssignees[assigneeID], topAssignees)
		}
	}
	if _, exists := topAssignees[assigneeFourID]; exists {
		t.Fatalf("expected assignee %d to be excluded from top 3, got %#v", assigneeFourID, topAssignees)
	}

	if stats.AvgCloseTimeSeconds == nil {
		t.Fatalf("expected average close time")
	}
	wantAvgCloseTimeSeconds := float64(16*3600) / 11
	if math.Abs(*stats.AvgCloseTimeSeconds-wantAvgCloseTimeSeconds) > 1 {
		t.Fatalf("expected average close time near %.2f seconds, got %.2f", wantAvgCloseTimeSeconds, *stats.AvgCloseTimeSeconds)
	}
	if stats.TotalComments != 4 {
		t.Fatalf("expected total comments 4, got %d", stats.TotalComments)
	}
}

func insertTeamStatsUser(t *testing.T, ctx context.Context, db *sqlx.DB, email string) uint64 {
	t.Helper()

	result, err := db.ExecContext(
		ctx,
		"INSERT INTO users (email, password_hash, name) VALUES (?, ?, ?)",
		email,
		"hash",
		"Stats Test User",
	)
	if err != nil {
		t.Fatalf("insert user %q: %v", email, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("get inserted user id: %v", err)
	}
	return uint64(id)
}

func insertTeamStatsTeam(t *testing.T, ctx context.Context, db *sqlx.DB, ownerID uint64) uint64 {
	t.Helper()

	result, err := db.ExecContext(
		ctx,
		"INSERT INTO teams (name, created_by) VALUES (?, ?)",
		"Stats Integration Team",
		ownerID,
	)
	if err != nil {
		t.Fatalf("insert team: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("get inserted team id: %v", err)
	}
	return uint64(id)
}

func insertTeamStatsTask(
	t *testing.T,
	ctx context.Context,
	db *sqlx.DB,
	teamID uint64,
	title string,
	status domain.TaskStatus,
	createdBy uint64,
	assigneeID uint64,
	createdHoursAgo int,
	closedHoursAgo int,
) uint64 {
	t.Helper()

	var result sql.Result
	var err error
	if closedHoursAgo == 0 {
		result, err = db.ExecContext(
			ctx,
			`INSERT INTO tasks (team_id, title, status, created_by, assignee_id)
			VALUES (?, ?, ?, ?, ?)`,
			teamID,
			title,
			status,
			createdBy,
			assigneeID,
		)
	} else {
		result, err = db.ExecContext(
			ctx,
			`INSERT INTO tasks (team_id, title, status, created_by, assignee_id, created_at, updated_at, closed_at)
			VALUES (?, ?, ?, ?, ?, UTC_TIMESTAMP() - INTERVAL ? HOUR, UTC_TIMESTAMP() - INTERVAL ? HOUR, UTC_TIMESTAMP() - INTERVAL ? HOUR)`,
			teamID,
			title,
			status,
			createdBy,
			assigneeID,
			createdHoursAgo,
			closedHoursAgo,
			closedHoursAgo,
		)
	}
	if err != nil {
		t.Fatalf("insert task %q: %v", title, err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("get inserted task id: %v", err)
	}
	return uint64(id)
}

func insertTeamStatsComment(t *testing.T, ctx context.Context, db *sqlx.DB, taskID uint64, userID uint64) {
	t.Helper()

	_, err := db.ExecContext(
		ctx,
		"INSERT INTO task_comments (task_id, user_id, content) VALUES (?, ?, ?)",
		taskID,
		userID,
		"stats integration comment",
	)
	if err != nil {
		t.Fatalf("insert comment: %v", err)
	}
}

func startTeamStatsMySQL(t *testing.T, ctx context.Context) *sqlx.DB {
	t.Helper()

	const (
		database = "task_management"
		username = "task_user"
		password = "task_password"
	)

	container, err := mysql.Run(
		ctx,
		"mysql:8.0.36",
		mysql.WithDatabase(database),
		mysql.WithUsername(username),
		mysql.WithPassword(password),
	)
	if err != nil {
		t.Fatalf("start mysql testcontainer: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := container.Terminate(cleanupCtx); err != nil {
			t.Fatalf("terminate mysql testcontainer: %v", err)
		}
	})

	dsn, err := container.ConnectionString(ctx, "parseTime=true", "multiStatements=true")
	if err != nil {
		t.Fatalf("build mysql testcontainer dsn: %v", err)
	}

	db, err := sqlx.ConnectContext(ctx, "mysql", dsn)
	if err != nil {
		t.Fatalf("connect to mysql testcontainer: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Fatalf("close mysql testcontainer connection: %v", err)
		}
	})

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping mysql testcontainer: %v", err)
	}

	applyTeamStatsMigrations(t, db.DB)

	return db
}

func applyTeamStatsMigrations(t *testing.T, db *sql.DB) {
	t.Helper()

	driver, err := migratemysql.WithInstance(db, &migratemysql.Config{
		DatabaseName: "task_management",
	})
	if err != nil {
		t.Fatalf("create mysql migrate driver: %v", err)
	}

	migrationsPath, err := teamStatsMigrationsPath()
	if err != nil {
		t.Fatalf("resolve migrations path: %v", err)
	}

	migrator, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", filepath.ToSlash(migrationsPath)),
		"mysql",
		driver,
	)
	if err != nil {
		t.Fatalf("create migrator: %v", err)
	}
	t.Cleanup(func() {
		sourceErr, databaseErr := migrator.Close()
		if sourceErr != nil {
			t.Fatalf("close migration source: %v", sourceErr)
		}
		if databaseErr != nil {
			t.Fatalf("close migration database: %v", databaseErr)
		}
	})

	if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		t.Fatalf("apply migrations: %v", err)
	}
}

func teamStatsMigrationsPath() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("runtime caller unavailable")
	}

	return filepath.Abs(filepath.Join(filepath.Dir(file), "..", "..", "..", "migrations"))
}
