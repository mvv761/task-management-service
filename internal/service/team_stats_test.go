package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"task-management-service/internal/domain"
)

type fakeTeamStatsRepository struct {
	stats *domain.TeamStats
	calls int
}

func (f *fakeTeamStatsRepository) GetByTeamID(ctx context.Context, teamID uint64) (*domain.TeamStats, error) {
	f.calls++
	return f.stats, nil
}

func TestTeamStatsService_GetTeamStats_RequiresOwnerOrAdmin(t *testing.T) {
	stats := &domain.TeamStats{TotalComments: 3}

	tests := []struct {
		name      string
		member    *domain.TeamMember
		memberErr error
		wantErr   error
		wantStats bool
	}{
		{
			name:      "owner can read stats",
			member:    &domain.TeamMember{TeamID: 1, UserID: 2, Role: domain.TeamRoleOwner},
			wantStats: true,
		},
		{
			name:      "admin can read stats",
			member:    &domain.TeamMember{TeamID: 1, UserID: 2, Role: domain.TeamRoleAdmin},
			wantStats: true,
		},
		{
			name:    "member cannot read stats",
			member:  &domain.TeamMember{TeamID: 1, UserID: 2, Role: domain.TeamRoleMember},
			wantErr: domain.ErrInsufficientPermissions,
		},
		{
			name:      "non-member cannot read stats",
			memberErr: domain.ErrTeamMemberNotFound,
			wantErr:   domain.ErrInsufficientPermissions,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeTeamStatsRepository{stats: stats}
			members := &fakeMemberFinder{
				members: map[uint64]*domain.TeamMember{},
			}
			if tt.member != nil {
				members.members[tt.member.UserID] = tt.member
			}
			if tt.memberErr != nil {
				members.err = tt.memberErr
			}

			got, err := NewTeamStatsService(repo, members).GetTeamStats(context.Background(), 2, 1)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			if tt.wantStats && !reflect.DeepEqual(got, stats) {
				t.Fatalf("expected stats %#v, got %#v", stats, got)
			}
			if !tt.wantStats && repo.calls != 0 {
				t.Fatalf("expected stats repository to be skipped")
			}
		})
	}
}
