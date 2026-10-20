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

type TaskCommentService interface {
	CreateComment(
		ctx context.Context,
		userID uint64,
		taskID uint64,
		content string,
	) (*domain.TaskComment, error)

	ListComments(
		ctx context.Context,
		userID uint64,
		taskID uint64,
	) ([]domain.TaskComment, error)
}

type TaskCommentHandler struct {
	service TaskCommentService
}

func NewTaskCommentHandler(service TaskCommentService) *TaskCommentHandler {
	return &TaskCommentHandler{
		service: service,
	}
}

func taskCommentToResponse(
	comment domain.TaskComment,
) dto.TaskCommentResponse {
	return dto.TaskCommentResponse{
		ID:        comment.ID,
		TaskID:    comment.TaskID,
		UserID:    comment.UserID,
		Content:   comment.Content,
		CreatedAt: comment.CreatedAt,
	}
}

// CreateComment creates a comment for a task.
// @Summary Create task comment
// @Description Добавить комментарий к задаче.
// @Tags comments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Task ID"
// @Param request body dto.CreateTaskCommentRequest true "Create comment request"
// @Success 201 {object} dto.TaskCommentResponse
// @Failure 400 {string} string "invalid request"
// @Failure 401 {string} string "unauthorized"
// @Failure 404 {string} string "task not found"
// @Failure 429 {string} string "rate limit exceeded"
// @Failure 500 {string} string "internal server error"
// @Router /api/v1/tasks/{id}/comments [post]
func (h *TaskCommentHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
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

	var req dto.CreateTaskCommentRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	comment, err := h.service.CreateComment(r.Context(), userID, taskID, req.Content)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrTaskNotFound),
			errors.Is(err, domain.ErrInsufficientPermissions):
			http.Error(w, domain.ErrTaskNotFound.Error(), http.StatusNotFound)

		case errors.Is(err, domain.ErrInvalidCommentContent):
			http.Error(w, domain.ErrInvalidCommentContent.Error(), http.StatusBadRequest)

		default:
			logHandlerError(r, "create task comment failed", err, "task_id", taskID)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	resp := taskCommentToResponse(*comment)

	if err := writeJSON(w, http.StatusCreated, resp); err != nil {
		logHandlerError(r, "write task comment response failed", err, "comment_id", comment.ID)
	}
}

// ListComments returns comments for a task.
// @Summary List task comments
// @Description Комментарии по задаче.
// @Tags comments
// @Produce json
// @Security BearerAuth
// @Param id path int true "Task ID"
// @Success 200 {object} dto.ListTaskCommentsResponse
// @Failure 400 {string} string "invalid task id"
// @Failure 401 {string} string "unauthorized"
// @Failure 404 {string} string "task not found"
// @Failure 429 {string} string "rate limit exceeded"
// @Failure 500 {string} string "internal server error"
// @Router /api/v1/tasks/{id}/comments [get]
func (h *TaskCommentHandler) ListComments(w http.ResponseWriter, r *http.Request) {
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

	comments, err := h.service.ListComments(r.Context(), userID, taskID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrTaskNotFound),
			errors.Is(err, domain.ErrInsufficientPermissions):
			http.Error(w, domain.ErrTaskNotFound.Error(), http.StatusNotFound)

		default:
			logHandlerError(r, "list task comments failed", err, "task_id", taskID)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	items := make([]dto.TaskCommentResponse, len(comments))

	for i, comment := range comments {
		items[i] = taskCommentToResponse(comment)
	}

	resp := dto.ListTaskCommentsResponse{
		Comments: items,
	}

	if err := writeJSON(w, http.StatusOK, resp); err != nil {
		logHandlerError(r, "write list task comments response failed", err, "task_id", taskID)
	}
}
