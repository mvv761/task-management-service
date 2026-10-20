package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"task-management-service/internal/domain"
	"task-management-service/internal/http/dto"
	appmiddleware "task-management-service/internal/http/middleware"
	"github.com/go-chi/chi/v5"
)

type TeamStatsService interface {
	GetTeamStats(
		ctx context.Context,
		userID uint64,
		teamID uint64,
	) (*domain.TeamStats, error)
}

type TeamStatsHandler struct {
	service TeamStatsService
}

func NewTeamStatsHandler(service TeamStatsService) *TeamStatsHandler {
	return &TeamStatsHandler{
		service: service,
	}
}

func teamStatsToResponse(stats domain.TeamStats) dto.TeamStatsResponse {
	resp := dto.TeamStatsResponse{
		ByStatus:            make([]dto.TaskStatusCountResponse, 0, len(stats.ByStatus)),
		TopAssignees:        make([]dto.TopAssigneeResponse, 0, len(stats.TopAssignees)),
		AvgCloseTimeSeconds: stats.AvgCloseTimeSeconds,
		TotalComments:       stats.TotalComments,
	}

	for _, item := range stats.ByStatus {
		resp.ByStatus = append(
			resp.ByStatus,
			dto.TaskStatusCountResponse{
				Status: string(item.Status),
				Count:  item.Count,
			},
		)
	}

	for _, item := range stats.TopAssignees {
		resp.TopAssignees = append(
			resp.TopAssignees,
			dto.TopAssigneeResponse{
				AssigneeID:  item.AssigneeID,
				ClosedCount: item.ClosedCount,
			},
		)
	}

	return resp
}

// GetTeamStats returns task statistics for a team.
// @Summary Get team statistics
// @Description Статистика по команде: статусы, топ исполнителей, среднее время закрытия, комментарии. Только owner/admin.
// @Tags stats
// @Produce json
// @Security BearerAuth
// @Param team_id path int true "Team ID"
// @Success 200 {object} dto.TeamStatsResponse
// @Failure 400 {string} string "invalid team id"
// @Failure 401 {string} string "unauthorized"
// @Failure 403 {string} string "insufficient permissions"
// @Failure 429 {string} string "rate limit exceeded"
// @Failure 500 {string} string "internal server error"
// @Router /api/v1/teams/{team_id}/stats [get]
func (h *TeamStatsHandler) GetTeamStats(w http.ResponseWriter, r *http.Request) {
	userID, ok := appmiddleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	teamIDParam := chi.URLParam(r, "team_id")

	teamID, err := strconv.ParseUint(teamIDParam, 10, 64)
	if err != nil || teamID == 0 {
		http.Error(w, "invalid team id", http.StatusBadRequest)
		return
	}

	stats, err := h.service.GetTeamStats(r.Context(), userID, teamID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInsufficientPermissions):
			http.Error(w, domain.ErrInsufficientPermissions.Error(), http.StatusForbidden)
		default:
			logHandlerError(r, "get team stats failed", err, "team_id", teamID)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	resp := teamStatsToResponse(*stats)

	if err := writeJSON(w, http.StatusOK, resp); err != nil {
		logHandlerError(r, "write team stats response failed", err, "team_id", teamID)
	}
}
