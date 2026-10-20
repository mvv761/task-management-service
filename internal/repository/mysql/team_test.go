package mysql

import (
	"context"
	"errors"
	"testing"
	"time"

	"task-management-service/internal/domain"
	"github.com/jmoiron/sqlx"
)

func TestTeamRepository_UpdateMemberRole_Integration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	db := startTeamStatsMySQL(t, ctx)
	repository := NewTeamRepository(db)

	ownerID := insertTeamStatsUser(t, ctx, db, "team-role-owner@example.test")
	memberID := insertTeamStatsUser(t, ctx, db, "team-role-member@example.test")
	teamID := insertTeamStatsTeam(t, ctx, db, ownerID)
	insertTeamMember(t, ctx, db, teamID, ownerID, domain.TeamRoleOwner)
	insertTeamMember(t, ctx, db, teamID, memberID, domain.TeamRoleMember)

	if err := repository.UpdateMemberRole(ctx, teamID, memberID, domain.TeamRoleAdmin); err != nil {
		t.Fatalf("UpdateMemberRole returned error: %v", err)
	}

	var role domain.TeamRole
	if err := db.GetContext(
		ctx,
		&role,
		"SELECT role FROM team_members WHERE team_id = ? AND user_id = ?",
		teamID,
		memberID,
	); err != nil {
		t.Fatalf("select updated team member role: %v", err)
	}

	if role != domain.TeamRoleAdmin {
		t.Fatalf("expected role %q, got %q", domain.TeamRoleAdmin, role)
	}

	err := repository.UpdateMemberRole(ctx, teamID, memberID+1000, domain.TeamRoleMember)
	if !errors.Is(err, domain.ErrTeamMemberNotFound) {
		t.Fatalf("expected ErrTeamMemberNotFound, got %v", err)
	}
}

func insertTeamMember(
	t *testing.T,
	ctx context.Context,
	db *sqlx.DB,
	teamID uint64,
	userID uint64,
	role domain.TeamRole,
) {
	t.Helper()

	_, err := db.ExecContext(
		ctx,
		"INSERT INTO team_members (team_id, user_id, role) VALUES (?, ?, ?)",
		teamID,
		userID,
		role,
	)
	if err != nil {
		t.Fatalf("insert team member: %v", err)
	}
}
