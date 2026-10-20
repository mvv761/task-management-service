package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"task-management-service/internal/domain"
	"github.com/go-chi/chi/v5"
)

type TeamMemberFinder interface {
	FindMember(
		ctx context.Context,
		teamID uint64,
		userID uint64,
	) (*domain.TeamMember, error)
}

type TeamAccess struct {
	memberFinder TeamMemberFinder
}

func NewTeamAccess(
	memberFinder TeamMemberFinder,
) *TeamAccess {
	return &TeamAccess{
		memberFinder: memberFinder,
	}
}

func (a *TeamAccess) RequireTeamMember(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := UserIDFromContext(r.Context())
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		teamIDParam := chi.URLParam(r, "id")

		teamID, err := strconv.ParseUint(teamIDParam, 10, 64)
		if err != nil {
			http.Error(w, "invalid team id", http.StatusBadRequest)
			return
		}

		_, err = a.memberFinder.FindMember(
			r.Context(),
			teamID,
			userID,
		)
		if err != nil {
			if errors.Is(err, domain.ErrTeamMemberNotFound) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			attrs := []any{
				"error", err,
				"team_id", teamID,
				"user_id", userID,
			}
			if requestID, ok := RequestIDFromContext(r.Context()); ok {
				attrs = append(attrs, "request_id", requestID)
			}

			slog.Default().Error("check team membership failed", attrs...)
			http.Error(
				w,
				"internal server error",
				http.StatusInternalServerError,
			)
			return
		}

		next.ServeHTTP(w, r)
	})
}
