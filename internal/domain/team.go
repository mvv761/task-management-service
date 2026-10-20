package domain

import (
	"errors"
	"time"
)

type TeamRole string

const (
	TeamRoleOwner  TeamRole = "owner"
	TeamRoleAdmin  TeamRole = "admin"
	TeamRoleMember TeamRole = "member"
)

var (
	ErrTeamNotFound            = errors.New("team not found")
	ErrInvalidTeamName         = errors.New("invalid team name")
	ErrTeamMemberNotFound      = errors.New("team member not found")
	ErrTeamMemberAlreadyExists = errors.New("team member already exists")
	ErrInsufficientPermissions = errors.New("insufficient permissions")
)

type Team struct {
	ID        uint64    `db:"id"`
	Name      string    `db:"name"`
	CreatedBy uint64    `db:"created_by"`
	CreatedAt time.Time `db:"created_at"`
}

type TeamMember struct {
	TeamID uint64   `db:"team_id"`
	UserID uint64   `db:"user_id"`
	Role   TeamRole `db:"role"`
}
