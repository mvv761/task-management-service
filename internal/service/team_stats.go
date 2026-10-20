package service

import (
	"context"
	"errors"
	"fmt"

	"task-management-service/internal/domain"
)

type TeamStatsRepository interface {
	GetByTeamID(
		ctx context.Context,
		teamID uint64,
	) (*domain.TeamStats, error)
}

type TeamStatsMemberFinder interface {
	FindMember(
		ctx context.Context,
		teamID uint64,
		userID uint64,
	) (*domain.TeamMember, error)
}

type TeamStatsService struct {
	statsRepository TeamStatsRepository
	memberFinder    TeamStatsMemberFinder
}

func NewTeamStatsService(
	statsRepository TeamStatsRepository,
	memberFinder TeamStatsMemberFinder,
) *TeamStatsService {
	return &TeamStatsService{
		statsRepository: statsRepository,
		memberFinder:    memberFinder,
	}
}

func (s *TeamStatsService) GetTeamStats(ctx context.Context, userID, teamID uint64) (*domain.TeamStats, error) {
	member, err := s.memberFinder.FindMember(ctx, teamID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrTeamMemberNotFound) {
			return nil, domain.ErrInsufficientPermissions
		}
		return nil, fmt.Errorf("find team member: %w", err)
	}

	if member.Role != domain.TeamRoleOwner &&
		member.Role != domain.TeamRoleAdmin {
		return nil, domain.ErrInsufficientPermissions
	}

	stats, err := s.statsRepository.GetByTeamID(
		ctx,
		teamID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"get team stats: %w",
			err,
		)
	}

	return stats, nil
}
