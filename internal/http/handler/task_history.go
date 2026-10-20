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

type TaskHistoryService interface {
	ListTaskHistory(ctx context.Context, userID uint64, taskID uint64) ([]domain.TaskHistory, error)
}

type TaskHistoryHandler struct {
	service TaskHistoryService
}

func NewTaskHistoryHandler(service TaskHistoryService) *TaskHistoryHandler {
	return &TaskHistoryHandler{
		service: service,
	}
}

func taskHistoryToResponse(history domain.TaskHistory) dto.TaskHistoryResponse {
	return dto.TaskHistoryResponse{
		ID:        history.ID,
		TaskID:    history.TaskID,
		ChangedBy: history.ChangedBy,
		Changes:   history.Changes,
		CreatedAt: history.CreatedAt,
	}
}

// ListTaskHistory returns change history for a task.
// @Summary List task history
// @Description История изменений задачи.
// @Tags history
// @Produce json
// @Security BearerAuth
// @Param id path int true "Task ID"
// @Success 200 {object} dto.ListTaskHistoryResponse
// @Failure 400 {string} string "invalid task id"
// @Failure 401 {string} string "unauthorized"
// @Failure 404 {string} string "task not found"
// @Failure 429 {string} string "rate limit exceeded"
// @Failure 500 {string} string "internal server error"
// @Router /api/v1/tasks/{id}/history [get]
func (h *TaskHistoryHandler) ListTaskHistory(w http.ResponseWriter, r *http.Request) {
	userID, ok := appmiddleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	taskIDParam := chi.URLParam(r, "id")

	taskID, err := strconv.ParseUint(taskIDParam, 10, 64)
	if err != nil || taskID == 0 {
		http.Error(w, "invalid task id", http.StatusBadRequest)
		return
	}

	history, err := h.service.ListTaskHistory(r.Context(), userID, taskID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrTaskNotFound),
			errors.Is(err, domain.ErrInsufficientPermissions):
			http.Error(
				w,
				domain.ErrTaskNotFound.Error(),
				http.StatusNotFound,
			)

		default:
			logHandlerError(r, "list task history failed", err, "task_id", taskID)

			http.Error(
				w,
				"internal server error",
				http.StatusInternalServerError,
			)
		}
		return
	}

	items := make(
		[]dto.TaskHistoryResponse,
		len(history),
	)

	for i, historyItem := range history {
		items[i] = taskHistoryToResponse(historyItem)
	}

	resp := dto.ListTaskHistoryResponse{
		History: items,
	}

	if err := writeJSON(
		w,
		http.StatusOK,
		resp,
	); err != nil {
		logHandlerError(r, "write task history response failed", err, "task_id", taskID)
	}
}
