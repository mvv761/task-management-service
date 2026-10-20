package service

import (
	"context"
	"errors"
	"testing"

	"task-management-service/internal/domain"
)

type fakeTeamRepository struct {
	memberFunc           func(ctx context.Context, teamID, userID uint64) (*domain.TeamMember, error)
	addMemberFunc        func(ctx context.Context, teamID, userID uint64, role domain.TeamRole) error
	updateMemberRoleFunc func(ctx context.Context, teamID, userID uint64, role domain.TeamRole) error

	addMemberCalls        int
	updateMemberRoleCalls int
	lastRole              domain.TeamRole
	lastUpdatedRole       domain.TeamRole
}

func (f *fakeTeamRepository) Create(ctx context.Context, team *domain.Team) error {
	team.ID = 1
	return nil
}

func (f *fakeTeamRepository) FindByID(ctx context.Context, id uint64) (*domain.Team, error) {
	return &domain.Team{ID: id, Name: "team"}, nil
}

func (f *fakeTeamRepository) ListByUserID(ctx context.Context, userID uint64) ([]domain.Team, error) {
	return nil, nil
}

func (f *fakeTeamRepository) FindMember(ctx context.Context, teamID, userID uint64) (*domain.TeamMember, error) {
	if f.memberFunc != nil {
		return f.memberFunc(ctx, teamID, userID)
	}
	return &domain.TeamMember{TeamID: teamID, UserID: userID, Role: domain.TeamRoleOwner}, nil
}

func (f *fakeTeamRepository) AddMember(ctx context.Context, teamID, userID uint64, role domain.TeamRole) error {
	f.addMemberCalls++
	f.lastRole = role
	if f.addMemberFunc != nil {
		return f.addMemberFunc(ctx, teamID, userID, role)
	}
	return nil
}

func (f *fakeTeamRepository) UpdateMemberRole(
	ctx context.Context,
	teamID uint64,
	userID uint64,
	role domain.TeamRole,
) error {
	f.updateMemberRoleCalls++
	f.lastUpdatedRole = role
	if f.updateMemberRoleFunc != nil {
		return f.updateMemberRoleFunc(ctx, teamID, userID, role)
	}
	return nil
}

type fakeUserFinder struct {
	err error
}

func (f *fakeUserFinder) FindByID(ctx context.Context, id uint64) (*domain.User, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &domain.User{ID: id, Email: "user@example.com", Name: "User"}, nil
}

func TestTeamService_InviteMember_TeamPermissions(t *testing.T) {
	tests := []struct {
		name           string
		actorRole      domain.TeamRole
		inviteRole     domain.TeamRole
		userErr        error
		wantErr        error
		wantAddMember  bool
		wantStoredRole domain.TeamRole
	}{
		{
			name:           "owner can invite member",
			actorRole:      domain.TeamRoleOwner,
			inviteRole:     domain.TeamRoleMember,
			wantAddMember:  true,
			wantStoredRole: domain.TeamRoleMember,
		},
		{
			name:           "admin can invite admin",
			actorRole:      domain.TeamRoleAdmin,
			inviteRole:     domain.TeamRoleAdmin,
			wantAddMember:  true,
			wantStoredRole: domain.TeamRoleAdmin,
		},
		{
			name:       "member cannot invite",
			actorRole:  domain.TeamRoleMember,
			inviteRole: domain.TeamRoleMember,
			wantErr:    domain.ErrInsufficientPermissions,
		},
		{
			name:       "cannot invite owner role",
			actorRole:  domain.TeamRoleOwner,
			inviteRole: domain.TeamRoleOwner,
			wantErr:    domain.ErrInvalidTeamRole,
		},
		{
			name:       "target user must exist",
			actorRole:  domain.TeamRoleOwner,
			inviteRole: domain.TeamRoleMember,
			userErr:    domain.ErrUserNotFound,
			wantErr:    domain.ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeTeamRepository{
				memberFunc: func(ctx context.Context, teamID, userID uint64) (*domain.TeamMember, error) {
					return &domain.TeamMember{TeamID: teamID, UserID: userID, Role: tt.actorRole}, nil
				},
			}
			users := &fakeUserFinder{err: tt.userErr}

			err := NewTeamService(repo, users).InviteMember(context.Background(), 1, 10, 2, tt.inviteRole)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			if got := repo.addMemberCalls > 0; got != tt.wantAddMember {
				t.Fatalf("expected add member call=%v, got calls=%d", tt.wantAddMember, repo.addMemberCalls)
			}
			if tt.wantAddMember && repo.lastRole != tt.wantStoredRole {
				t.Fatalf("expected stored role %q, got %q", tt.wantStoredRole, repo.lastRole)
			}
		})
	}
}

func TestTeamService_InviteMember_NonMemberActorDenied(t *testing.T) {
	repo := &fakeTeamRepository{
		memberFunc: func(ctx context.Context, teamID, userID uint64) (*domain.TeamMember, error) {
			return nil, domain.ErrTeamMemberNotFound
		},
	}

	err := NewTeamService(repo, &fakeUserFinder{}).InviteMember(context.Background(), 1, 10, 2, domain.TeamRoleMember)
	if !errors.Is(err, domain.ErrInsufficientPermissions) {
		t.Fatalf("expected ErrInsufficientPermissions, got %v", err)
	}
	if repo.addMemberCalls != 0 {
		t.Fatalf("expected AddMember to be skipped")
	}
}

func TestTeamService_UpdateMemberRole(t *testing.T) {
	ctx := context.Background()
	const (
		actorUserID  = uint64(1)
		targetUserID = uint64(2)
		teamID       = uint64(10)
	)

	errDatabaseUnavailable := errors.New("database unavailable")

	tests := []struct {
		name            string
		actorRole       domain.TeamRole
		actorErr        error
		targetRole      domain.TeamRole
		targetErr       error
		newRole         domain.TeamRole
		updateErr       error
		wantErr         error
		wantUpdate      bool
		wantUpdatedRole domain.TeamRole
	}{
		{
			name:            "owner changes member to admin",
			actorRole:       domain.TeamRoleOwner,
			targetRole:      domain.TeamRoleMember,
			newRole:         domain.TeamRoleAdmin,
			wantUpdate:      true,
			wantUpdatedRole: domain.TeamRoleAdmin,
		},
		{
			name:            "owner changes admin to member",
			actorRole:       domain.TeamRoleOwner,
			targetRole:      domain.TeamRoleAdmin,
			newRole:         domain.TeamRoleMember,
			wantUpdate:      true,
			wantUpdatedRole: domain.TeamRoleMember,
		},
		{
			name:       "admin cannot change member role",
			actorRole:  domain.TeamRoleAdmin,
			targetRole: domain.TeamRoleMember,
			newRole:    domain.TeamRoleAdmin,
			wantErr:    domain.ErrInsufficientPermissions,
		},
		{
			name:       "member cannot change role",
			actorRole:  domain.TeamRoleMember,
			targetRole: domain.TeamRoleMember,
			newRole:    domain.TeamRoleAdmin,
			wantErr:    domain.ErrInsufficientPermissions,
		},
		{
			name:     "non-member actor denied",
			actorErr: domain.ErrTeamMemberNotFound,
			newRole:  domain.TeamRoleAdmin,
			wantErr:  domain.ErrInsufficientPermissions,
		},
		{
			name:       "target user not member",
			actorRole:  domain.TeamRoleOwner,
			targetRole: domain.TeamRoleMember,
			targetErr:  domain.ErrTeamMemberNotFound,
			newRole:    domain.TeamRoleAdmin,
			wantErr:    domain.ErrTeamMemberNotFound,
		},
		{
			name:       "cannot assign owner role",
			actorRole:  domain.TeamRoleOwner,
			targetRole: domain.TeamRoleMember,
			newRole:    domain.TeamRoleOwner,
			wantErr:    domain.ErrInvalidTeamRole,
		},
		{
			name:       "unknown role rejected",
			actorRole:  domain.TeamRoleOwner,
			targetRole: domain.TeamRoleMember,
			newRole:    domain.TeamRole("manager"),
			wantErr:    domain.ErrInvalidTeamRole,
		},
		{
			name:       "cannot change current owner role",
			actorRole:  domain.TeamRoleOwner,
			targetRole: domain.TeamRoleOwner,
			newRole:    domain.TeamRoleMember,
			wantErr:    domain.ErrInsufficientPermissions,
		},
		{
			name:       "same role update is idempotent",
			actorRole:  domain.TeamRoleOwner,
			targetRole: domain.TeamRoleMember,
			newRole:    domain.TeamRoleMember,
		},
		{
			name:            "repository update infrastructure error is wrapped",
			actorRole:       domain.TeamRoleOwner,
			targetRole:      domain.TeamRoleMember,
			newRole:         domain.TeamRoleAdmin,
			updateErr:       errDatabaseUnavailable,
			wantErr:         errDatabaseUnavailable,
			wantUpdate:      true,
			wantUpdatedRole: domain.TeamRoleAdmin,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeTeamRepository{
				memberFunc: func(ctx context.Context, gotTeamID, gotUserID uint64) (*domain.TeamMember, error) {
					if gotTeamID != teamID {
						t.Fatalf("expected team ID %d, got %d", teamID, gotTeamID)
					}

					switch gotUserID {
					case actorUserID:
						if tt.actorErr != nil {
							return nil, tt.actorErr
						}
						return &domain.TeamMember{
							TeamID: teamID,
							UserID: actorUserID,
							Role:   tt.actorRole,
						}, nil

					case targetUserID:
						if tt.targetErr != nil {
							return nil, tt.targetErr
						}
						return &domain.TeamMember{
							TeamID: teamID,
							UserID: targetUserID,
							Role:   tt.targetRole,
						}, nil

					default:
						t.Fatalf("unexpected user ID %d", gotUserID)
						return nil, nil
					}
				},
				updateMemberRoleFunc: func(ctx context.Context, gotTeamID, gotUserID uint64, role domain.TeamRole) error {
					if gotTeamID != teamID {
						t.Fatalf("expected update team ID %d, got %d", teamID, gotTeamID)
					}
					if gotUserID != targetUserID {
						t.Fatalf("expected update user ID %d, got %d", targetUserID, gotUserID)
					}
					return tt.updateErr
				},
			}

			err := NewTeamService(repo, &fakeUserFinder{}).UpdateMemberRole(
				ctx,
				actorUserID,
				teamID,
				targetUserID,
				tt.newRole,
			)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
			if got := repo.updateMemberRoleCalls > 0; got != tt.wantUpdate {
				t.Fatalf("expected update call=%v, got calls=%d", tt.wantUpdate, repo.updateMemberRoleCalls)
			}
			if tt.wantUpdate && repo.updateMemberRoleCalls != 1 {
				t.Fatalf("expected one update call, got %d", repo.updateMemberRoleCalls)
			}
			if tt.wantUpdate && repo.lastUpdatedRole != tt.wantUpdatedRole {
				t.Fatalf("expected updated role %q, got %q", tt.wantUpdatedRole, repo.lastUpdatedRole)
			}
		})
	}
}
