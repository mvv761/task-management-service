package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"task-management-service/internal/domain"
	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

type TeamRepository struct {
	db *sqlx.DB
}

func NewTeamRepository(db *sqlx.DB) *TeamRepository {
	return &TeamRepository{db: db}
}

func (r *TeamRepository) Create(
	ctx context.Context,
	team *domain.Team,
) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer tx.Rollback()

	teamQuery := `
		INSERT INTO teams (name, created_by)
		VALUES (?, ?)
	`

	res, err := tx.ExecContext(
		ctx,
		teamQuery,
		team.Name,
		team.CreatedBy,
	)
	if err != nil {
		return fmt.Errorf("create team: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("get team id: %w", err)
	}

	teamID := uint64(id)

	ownerQuery := `
		INSERT INTO team_members (team_id, user_id, role)
		VALUES (?, ?, ?)
	`

	_, err = tx.ExecContext(
		ctx,
		ownerQuery,
		teamID,
		team.CreatedBy,
		domain.TeamRoleOwner,
	)
	if err != nil {
		return fmt.Errorf("create team owner: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	team.ID = teamID

	return nil
}

func (r *TeamRepository) FindByID(ctx context.Context, id uint64) (*domain.Team, error) {
	var team domain.Team
	query := `SELECT id, name, created_by, created_at FROM teams WHERE id = ?`
	err := r.db.GetContext(ctx, &team, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrTeamNotFound
		}

		return nil, fmt.Errorf("find team by id: %w", err)
	}

	return &team, nil
}

func (r *TeamRepository) ListByUserID(ctx context.Context, userID uint64) ([]domain.Team, error) {
	teams := make([]domain.Team, 0)
	query := `
		SELECT t.id, t.name, t.created_by, t.created_at
		FROM teams t
		JOIN team_members tm ON t.id = tm.team_id
		WHERE tm.user_id = ?
	`

	err := r.db.SelectContext(ctx, &teams, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list teams by user id: %w", err)
	}

	return teams, nil
}

func (r *TeamRepository) FindMember(ctx context.Context, teamID, userID uint64) (*domain.TeamMember, error) {
	var member domain.TeamMember

	query := `SELECT team_id, user_id, role FROM team_members WHERE team_id = ? AND user_id = ?`

	err := r.db.GetContext(ctx, &member, query, teamID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrTeamMemberNotFound
		}
		return nil, fmt.Errorf("find team member: %w", err)
	}

	return &member, nil
}

func (r *TeamRepository) AddMember(ctx context.Context, teamID, userID uint64, role domain.TeamRole) error {
	query := `INSERT INTO team_members (team_id, user_id, role) VALUES (?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, teamID, userID, role)
	if err != nil {
		if mysqlErr, ok := errors.AsType[*mysqldriver.MySQLError](err); ok &&
			mysqlErr.Number == mysqlDuplicateEntryError {
			return domain.ErrTeamMemberAlreadyExists
		}

		return fmt.Errorf("add team member: %w", err)
	}

	return nil
}

func (r *TeamRepository) UpdateMemberRole(
	ctx context.Context,
	teamID uint64,
	userID uint64,
	role domain.TeamRole,
) error {
	query := `
		UPDATE team_members
		SET role = ?
		WHERE team_id = ?
		  AND user_id = ?
	`

	res, err := r.db.ExecContext(ctx, query, role, teamID, userID)
	if err != nil {
		return fmt.Errorf("update team member role: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("get updated team member role rows affected: %w", err)
	}

	if affected == 0 {
		return domain.ErrTeamMemberNotFound
	}

	return nil
}
