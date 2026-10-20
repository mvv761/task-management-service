package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"task-management-service/internal/domain"
	appmiddleware "task-management-service/internal/http/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
)

type fakeTeamHandlerService struct {
	updateMemberRoleErr error

	updateMemberRoleCalls int
	actorUserID           uint64
	teamID                uint64
	targetUserID          uint64
	role                  domain.TeamRole
}

func (f *fakeTeamHandlerService) CreateTeam(
	ctx context.Context,
	userID uint64,
	name string,
) (*domain.Team, error) {
	return &domain.Team{ID: 1, Name: name, CreatedBy: userID}, nil
}

func (f *fakeTeamHandlerService) ListTeams(ctx context.Context, userID uint64) ([]domain.Team, error) {
	return nil, nil
}

func (f *fakeTeamHandlerService) InviteMember(
	ctx context.Context,
	actorUserID uint64,
	teamID uint64,
	targetUserID uint64,
	role domain.TeamRole,
) error {
	return nil
}

func (f *fakeTeamHandlerService) UpdateMemberRole(
	ctx context.Context,
	actorUserID uint64,
	teamID uint64,
	targetUserID uint64,
	role domain.TeamRole,
) error {
	f.updateMemberRoleCalls++
	f.actorUserID = actorUserID
	f.teamID = teamID
	f.targetUserID = targetUserID
	f.role = role
	return f.updateMemberRoleErr
}

func TestTeamHandler_UpdateMemberRole(t *testing.T) {
	const (
		secret       = "test-secret"
		actorUserID  = uint64(7)
		teamID       = uint64(10)
		targetUserID = uint64(2)
	)

	tests := []struct {
		name          string
		path          string
		body          string
		serviceErr    error
		wantStatus    int
		wantCall      bool
		wantRole      domain.TeamRole
		wantEmptyBody bool
	}{
		{
			name:          "success",
			path:          "/api/v1/teams/10/members/2",
			body:          `{"role":"admin"}`,
			wantStatus:    http.StatusNoContent,
			wantCall:      true,
			wantRole:      domain.TeamRoleAdmin,
			wantEmptyBody: true,
		},
		{
			name:       "invalid role",
			path:       "/api/v1/teams/10/members/2",
			body:       `{"role":"owner"}`,
			serviceErr: domain.ErrInvalidTeamRole,
			wantStatus: http.StatusBadRequest,
			wantCall:   true,
			wantRole:   domain.TeamRoleOwner,
		},
		{
			name:       "permission error",
			path:       "/api/v1/teams/10/members/2",
			body:       `{"role":"admin"}`,
			serviceErr: domain.ErrInsufficientPermissions,
			wantStatus: http.StatusForbidden,
			wantCall:   true,
			wantRole:   domain.TeamRoleAdmin,
		},
		{
			name:       "target not found",
			path:       "/api/v1/teams/10/members/2",
			body:       `{"role":"admin"}`,
			serviceErr: domain.ErrTeamMemberNotFound,
			wantStatus: http.StatusNotFound,
			wantCall:   true,
			wantRole:   domain.TeamRoleAdmin,
		},
		{
			name:       "invalid path id",
			path:       "/api/v1/teams/0/members/2",
			body:       `{"role":"admin"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid json",
			path:       "/api/v1/teams/10/members/2",
			body:       `{"role":`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeTeamHandlerService{
				updateMemberRoleErr: tt.serviceErr,
			}
			router := chi.NewRouter()
			router.Use(appmiddleware.NewAuth(secret).Authenticate)
			router.Patch("/api/v1/teams/{team_id}/members/{user_id}", NewTeamHandler(service).UpdateMemberRole)

			req := httptest.NewRequest(http.MethodPatch, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, actorUserID))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d with body %q", tt.wantStatus, rr.Code, rr.Body.String())
			}
			if tt.wantEmptyBody && rr.Body.Len() != 0 {
				t.Fatalf("expected empty response body, got %q", rr.Body.String())
			}
			if got := service.updateMemberRoleCalls > 0; got != tt.wantCall {
				t.Fatalf("expected service call=%v, got calls=%d", tt.wantCall, service.updateMemberRoleCalls)
			}
			if !tt.wantCall {
				return
			}
			if service.updateMemberRoleCalls != 1 {
				t.Fatalf("expected one service call, got %d", service.updateMemberRoleCalls)
			}
			if service.actorUserID != actorUserID {
				t.Fatalf("expected actor user ID %d, got %d", actorUserID, service.actorUserID)
			}
			if service.teamID != teamID {
				t.Fatalf("expected team ID %d, got %d", teamID, service.teamID)
			}
			if service.targetUserID != targetUserID {
				t.Fatalf("expected target user ID %d, got %d", targetUserID, service.targetUserID)
			}
			if service.role != tt.wantRole {
				t.Fatalf("expected role %q, got %q", tt.wantRole, service.role)
			}
		})
	}
}

func testJWT(t *testing.T, secret string, userID uint64) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject: strconv.FormatUint(userID, 10),
	})

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign test jwt: %v", err)
	}

	if tokenString == "" {
		t.Fatalf("invalid test jwt")
	}

	return tokenString
}
