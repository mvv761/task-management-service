package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"task-management-service/internal/domain"
	"task-management-service/internal/http/dto"
	appmiddleware "task-management-service/internal/http/middleware"
	"github.com/go-chi/chi/v5"
)

type TeamService interface {
	CreateTeam(
		ctx context.Context,
		userID uint64,
		name string,
	) (*domain.Team, error)

	ListTeams(
		ctx context.Context,
		userID uint64,
	) ([]domain.Team, error)

	InviteMember(
		ctx context.Context,
		actorUserID uint64,
		teamID uint64,
		targetUserID uint64,
		role domain.TeamRole,
	) error

	UpdateMemberRole(
		ctx context.Context,
		actorUserID uint64,
		teamID uint64,
		targetUserID uint64,
		role domain.TeamRole,
	) error
}

type TeamHandler struct {
	service TeamService
}

func NewTeamHandler(service TeamService) *TeamHandler {
	return &TeamHandler{
		service: service,
	}
}

// CreateTeam creates a team owned by the current user.
// @Summary Create team
// @Description Создать команду — текущий пользователь становится owner.
// @Tags teams
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.CreateTeamRequest true "Create team request"
// @Success 201 {object} dto.TeamResponse
// @Failure 400 {string} string "invalid request"
// @Failure 401 {string} string "unauthorized"
// @Failure 429 {string} string "rate limit exceeded"
// @Failure 500 {string} string "internal server error"
// @Router /api/v1/teams [post]
func (h *TeamHandler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	userID, ok := appmiddleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req dto.CreateTeamRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	team, err := h.service.CreateTeam(r.Context(), userID, req.Name)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidTeamName) {
			http.Error(
				w,
				err.Error(),
				http.StatusBadRequest,
			)
			return
		}

		logHandlerError(r, "create team failed", err)
		http.Error(
			w,
			"internal server error",
			http.StatusInternalServerError,
		)
		return
	}

	resp := dto.TeamResponse{
		ID:        team.ID,
		Name:      team.Name,
		CreatedBy: team.CreatedBy,
	}

	if err := writeJSON(w, http.StatusCreated, resp); err != nil {
		logHandlerError(r, "write create team response failed", err, "team_id", team.ID)
	}
}

// ListTeams returns teams available to the current user.
// @Summary List teams
// @Description Команды, в которых состоит текущий пользователь.
// @Tags teams
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dto.ListTeamsResponse
// @Failure 401 {string} string "unauthorized"
// @Failure 429 {string} string "rate limit exceeded"
// @Failure 500 {string} string "internal server error"
// @Router /api/v1/teams [get]
func (h *TeamHandler) ListTeams(w http.ResponseWriter, r *http.Request) {
	userID, ok := appmiddleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	teams, err := h.service.ListTeams(r.Context(), userID)
	if err != nil {
		logHandlerError(r, "list teams failed", err)
		http.Error(
			w,
			"internal server error",
			http.StatusInternalServerError,
		)
		return
	}

	resp := dto.ListTeamsResponse{
		Teams: make([]dto.TeamResponse, 0, len(teams)),
	}

	for _, team := range teams {
		resp.Teams = append(resp.Teams, dto.TeamResponse{
			ID:        team.ID,
			Name:      team.Name,
			CreatedBy: team.CreatedBy,
		})
	}

	if err := writeJSON(w, http.StatusOK, resp); err != nil {
		logHandlerError(r, "write list teams response failed", err)
	}
}

// InviteMember adds a user to a team.
// @Summary Invite team member
// @Description Пригласить в команду (роль admin/member). Доступно owner и admin.
// @Tags teams
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Team ID"
// @Param request body dto.InviteMemberRequest true "Invite member request"
// @Success 204
// @Failure 400 {string} string "invalid request"
// @Failure 401 {string} string "unauthorized"
// @Failure 403 {string} string "insufficient permissions"
// @Failure 404 {string} string "user not found"
// @Failure 409 {string} string "team member already exists"
// @Failure 429 {string} string "rate limit exceeded"
// @Failure 500 {string} string "internal server error"
// @Router /api/v1/teams/{id}/invite [post]
func (h *TeamHandler) InviteMember(w http.ResponseWriter, r *http.Request) {
	actorUserID, ok := appmiddleware.UserIDFromContext(r.Context())
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

	var req dto.InviteMemberRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	err = h.service.InviteMember(r.Context(), actorUserID, teamID, req.UserID, domain.TeamRole(req.Role))
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidTeamRole):
			http.Error(
				w,
				domain.ErrInvalidTeamRole.Error(),
				http.StatusBadRequest,
			)

		case errors.Is(err, domain.ErrUserNotFound):
			http.Error(
				w,
				domain.ErrUserNotFound.Error(),
				http.StatusNotFound,
			)

		case errors.Is(err, domain.ErrInsufficientPermissions):
			http.Error(
				w,
				domain.ErrInsufficientPermissions.Error(),
				http.StatusForbidden,
			)

		case errors.Is(err, domain.ErrTeamMemberAlreadyExists):
			http.Error(
				w,
				domain.ErrTeamMemberAlreadyExists.Error(),
				http.StatusConflict,
			)

		default:
			logHandlerError(
				r,
				"invite team member failed",
				err,
				"team_id", teamID,
				"target_user_id", req.UserID,
			)
			http.Error(
				w,
				"internal server error",
				http.StatusInternalServerError,
			)
		}

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateMemberRole updates an existing team member role.
// @Summary Update team member role
// @Description Сменить роль участника (admin/member). Owner через этот метод не назначается.
// @Tags teams
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param team_id path int true "Team ID"
// @Param user_id path int true "User ID"
// @Param request body dto.UpdateTeamMemberRoleRequest true "Update team member role request"
// @Success 204
// @Failure 400 {string} string "invalid request or role"
// @Failure 401 {string} string "unauthorized"
// @Failure 403 {string} string "insufficient permissions"
// @Failure 404 {string} string "team member not found"
// @Failure 429 {string} string "rate limit exceeded"
// @Failure 500 {string} string "internal server error"
// @Router /api/v1/teams/{team_id}/members/{user_id} [patch]
func (h *TeamHandler) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	actorUserID, ok := appmiddleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	teamID, err := parsePositiveUintParam(r, "team_id", "invalid team id")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	targetUserID, err := parsePositiveUintParam(r, "user_id", "invalid user id")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var req dto.UpdateTeamMemberRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	err = h.service.UpdateMemberRole(
		r.Context(),
		actorUserID,
		teamID,
		targetUserID,
		domain.TeamRole(req.Role),
	)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidTeamRole):
			http.Error(w, domain.ErrInvalidTeamRole.Error(), http.StatusBadRequest)

		case errors.Is(err, domain.ErrInsufficientPermissions):
			http.Error(w, domain.ErrInsufficientPermissions.Error(), http.StatusForbidden)

		case errors.Is(err, domain.ErrTeamMemberNotFound):
			http.Error(w, domain.ErrTeamMemberNotFound.Error(), http.StatusNotFound)

		default:
			logHandlerError(
				r,
				"update team member role failed",
				err,
				"team_id", teamID,
				"target_user_id", targetUserID,
				"actor_user_id", actorUserID,
			)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func parsePositiveUintParam(r *http.Request, name string, message string) (uint64, error) {
	value, err := strconv.ParseUint(chi.URLParam(r, name), 10, 64)
	if err != nil || value == 0 {
		return 0, errors.New(message)
	}

	return value, nil
}
