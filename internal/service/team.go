package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"task-management-service/internal/domain"
)

type TeamRepository interface {
	Create(ctx context.Context, team *domain.Team) error
	FindByID(ctx context.Context, id uint64) (*domain.Team, error)
	ListByUserID(ctx context.Context, userID uint64) ([]domain.Team, error)
	FindMember(ctx context.Context, teamID, userID uint64) (*domain.TeamMember, error)
	AddMember(ctx context.Context, teamID, userID uint64, role domain.TeamRole) error
	UpdateMemberRole(ctx context.Context, teamID, userID uint64, role domain.TeamRole) error
}

type UserFinder interface {
	FindByID(
		ctx context.Context,
		id uint64,
	) (*domain.User, error)
}
type TeamService struct {
	teamRepository TeamRepository
	userFinder     UserFinder
}

func NewTeamService(teamRepository TeamRepository, userFinder UserFinder) *TeamService {
	return &TeamService{
		teamRepository: teamRepository,
		userFinder:     userFinder,
	}
}

func (s *TeamService) CreateTeam(ctx context.Context, userID uint64, name string) (*domain.Team, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, domain.ErrInvalidTeamName
	}

	team := &domain.Team{
		Name:      name,
		CreatedBy: userID,
	}

	if err := s.teamRepository.Create(ctx, team); err != nil {
		return nil, fmt.Errorf("create team: %w", err)
	}

	return team, nil
}

func (s *TeamService) ListTeams(ctx context.Context, userID uint64) ([]domain.Team, error) {
	teams, err := s.teamRepository.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list teams: %w", err)
	}

	return teams, nil
}

func (s *TeamService) InviteMember(
	ctx context.Context,
	actorUserID uint64,
	teamID uint64,
	targetUserID uint64,
	role domain.TeamRole,
) error {
	actor, err := s.teamRepository.FindMember(ctx, teamID, actorUserID)
	if err != nil {
		if errors.Is(err, domain.ErrTeamMemberNotFound) {
			return domain.ErrInsufficientPermissions
		}
		return fmt.Errorf("find actor member: %w", err)
	}

	if actor.Role != domain.TeamRoleOwner && actor.Role != domain.TeamRoleAdmin {
		return domain.ErrInsufficientPermissions
	}

	if role != domain.TeamRoleAdmin &&
		role != domain.TeamRoleMember {
		return domain.ErrInvalidTeamRole
	}

	_, err = s.userFinder.FindByID(ctx, targetUserID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return domain.ErrUserNotFound
		}
		return fmt.Errorf("find target user: %w", err)
	}

	if err := s.teamRepository.AddMember(ctx, teamID, targetUserID, role); err != nil {
		return fmt.Errorf("add member: %w", err)
	}

	return nil
}

func (s *TeamService) UpdateMemberRole(
	ctx context.Context,
	actorUserID uint64,
	teamID uint64,
	targetUserID uint64,
	role domain.TeamRole,
) error {
	if role != domain.TeamRoleAdmin && role != domain.TeamRoleMember {
		return domain.ErrInvalidTeamRole
	}

	actor, err := s.teamRepository.FindMember(ctx, teamID, actorUserID)
	if err != nil {
		if errors.Is(err, domain.ErrTeamMemberNotFound) {
			return domain.ErrInsufficientPermissions
		}
		return fmt.Errorf("find actor member: %w", err)
	}

	if actor.Role != domain.TeamRoleOwner {
		return domain.ErrInsufficientPermissions
	}

	target, err := s.teamRepository.FindMember(ctx, teamID, targetUserID)
	if err != nil {
		if errors.Is(err, domain.ErrTeamMemberNotFound) {
			return domain.ErrTeamMemberNotFound
		}
		return fmt.Errorf("find target member: %w", err)
	}

	if target.Role == domain.TeamRoleOwner {
		return domain.ErrInsufficientPermissions
	}

	if target.Role == role {
		return nil
	}

	if err := s.teamRepository.UpdateMemberRole(ctx, teamID, targetUserID, role); err != nil {
		return fmt.Errorf("update member role: %w", err)
	}

	return nil
}
