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

type TaskService interface {
	CreateTask(
		ctx context.Context,
		teamID uint64,
		userID uint64,
		title string,
		description *string,
		assigneeID *uint64,
	) (*domain.Task, error)
	ListTasks(
		ctx context.Context,
		userID uint64,
		filter domain.TaskFilter,
	) ([]domain.Task, error)
	GetTask(
		ctx context.Context,
		userID uint64,
		taskID uint64,
	) (*domain.Task, error)
	UpdateTask(
		ctx context.Context,
		userID uint64,
		taskID uint64,
		patch domain.TaskPatch,
	) (*domain.Task, error)
}

type TaskHandler struct {
	service TaskService
}

func NewTaskHandler(service TaskService) *TaskHandler {
	return &TaskHandler{
		service: service,
	}
}

// CreateTask creates a task in a team.
// @Summary Create task
// @Description Создать задачу в команде. Исполнитель — только участник этой же команды.
// @Tags tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.CreateTaskRequest true "Create task request"
// @Success 201 {object} dto.TaskResponse
// @Failure 400 {string} string "invalid request"
// @Failure 401 {string} string "unauthorized"
// @Failure 403 {string} string "insufficient permissions"
// @Failure 429 {string} string "rate limit exceeded"
// @Failure 500 {string} string "internal server error"
// @Router /api/v1/tasks [post]
func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	userID, ok := appmiddleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req dto.CreateTaskRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	task, err := h.service.CreateTask(r.Context(), req.TeamID, userID, req.Title, req.Description, req.AssigneeID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidTaskTitle):
			http.Error(
				w,
				domain.ErrInvalidTaskTitle.Error(),
				http.StatusBadRequest,
			)

		case errors.Is(err, domain.ErrInvalidAssignee):
			http.Error(
				w,
				domain.ErrInvalidAssignee.Error(),
				http.StatusBadRequest,
			)

		case errors.Is(err, domain.ErrInsufficientPermissions):
			http.Error(
				w,
				domain.ErrInsufficientPermissions.Error(),
				http.StatusForbidden,
			)

		default:
			logHandlerError(r, "create task failed", err, "team_id", req.TeamID)
			http.Error(
				w,
				"internal server error",
				http.StatusInternalServerError,
			)
		}

		return
	}

	resp := taskToResponse(*task)

	if err := writeJSON(w, http.StatusCreated, resp); err != nil {
		logHandlerError(r, "write create task response failed", err, "task_id", task.ID)
	}
}

// ListTasks returns tasks by team with optional filters.
// @Summary List tasks
// @Description Список задач команды: фильтры status/assignee_id, пагинация limit/offset. Ответ кешируется.
// @Tags tasks
// @Produce json
// @Security BearerAuth
// @Param team_id query int true "Team ID"
// @Param status query string false "Task status" Enums(todo,in_progress,done,cancelled)
// @Param assignee_id query int false "Assignee user ID"
// @Param limit query int false "Page size"
// @Param offset query int false "Page offset"
// @Success 200 {object} dto.ListTasksResponse
// @Failure 400 {string} string "invalid query parameter"
// @Failure 401 {string} string "unauthorized"
// @Failure 403 {string} string "insufficient permissions"
// @Failure 429 {string} string "rate limit exceeded"
// @Failure 500 {string} string "internal server error"
// @Router /api/v1/tasks [get]
func (h *TaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	userID, ok := appmiddleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	teamIDParam := r.URL.Query().Get("team_id")

	teamID, err := strconv.ParseUint(teamIDParam, 10, 64)
	if err != nil || teamID == 0 {
		http.Error(w, "invalid team id", http.StatusBadRequest)
		return
	}

	var status *domain.TaskStatus

	statusParam := r.URL.Query().Get("status")
	if statusParam != "" {
		taskStatus := domain.TaskStatus(statusParam)
		status = &taskStatus
	}

	var assigneeID *uint64

	assigneeIDParam := r.URL.Query().Get("assignee_id")
	if assigneeIDParam != "" {
		parsedAssigneeID, err := strconv.ParseUint(assigneeIDParam, 10, 64)
		if err != nil || parsedAssigneeID == 0 {
			http.Error(w, "invalid assignee id", http.StatusBadRequest)
			return
		}
		assigneeID = &parsedAssigneeID
	}

	var limit uint32

	limitParam := r.URL.Query().Get("limit")
	if limitParam != "" {
		parsedLimit, err := strconv.ParseUint(limitParam, 10, 32)
		if err != nil || parsedLimit == 0 {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		limit = uint32(parsedLimit)
	}

	var offset uint32

	offsetParam := r.URL.Query().Get("offset")
	if offsetParam != "" {
		parsedOffset, err := strconv.ParseUint(offsetParam, 10, 32)
		if err != nil {
			http.Error(w, "invalid offset", http.StatusBadRequest)
			return
		}
		offset = uint32(parsedOffset)
	}

	filter := domain.TaskFilter{
		TeamID:     teamID,
		Status:     status,
		AssigneeID: assigneeID,
		Limit:      limit,
		Offset:     offset,
	}

	tasks, err := h.service.ListTasks(
		r.Context(),
		userID,
		filter,
	)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidTaskStatus):
			http.Error(
				w,
				domain.ErrInvalidTaskStatus.Error(),
				http.StatusBadRequest,
			)

		case errors.Is(err, domain.ErrInsufficientPermissions):
			http.Error(
				w,
				domain.ErrInsufficientPermissions.Error(),
				http.StatusForbidden,
			)

		default:
			logHandlerError(r, "list tasks failed", err, "team_id", teamID)
			http.Error(
				w,
				"internal server error",
				http.StatusInternalServerError,
			)
		}

		return
	}

	resp := dto.ListTasksResponse{
		Tasks: make([]dto.TaskResponse, 0, len(tasks)),
	}

	for _, task := range tasks {
		resp.Tasks = append(
			resp.Tasks,
			taskToResponse(task),
		)
	}

	if err := writeJSON(w, http.StatusOK, resp); err != nil {
		logHandlerError(r, "write list tasks response failed", err, "team_id", teamID)
	}
}

func taskToResponse(task domain.Task) dto.TaskResponse {
	return dto.TaskResponse{
		ID:          task.ID,
		TeamID:      task.TeamID,
		Title:       task.Title,
		Description: task.Description,
		Status:      string(task.Status),
		CreatedBy:   task.CreatedBy,
		AssigneeID:  task.AssigneeID,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
		ClosedAt:    task.ClosedAt,
		Version:     task.Version,
	}
}

// GetTask returns a task by ID.
// @Summary Get task
// @Description Получить задачу по id (только участники её команды).
// @Tags tasks
// @Produce json
// @Security BearerAuth
// @Param id path int true "Task ID"
// @Success 200 {object} dto.TaskResponse
// @Failure 400 {string} string "invalid task id"
// @Failure 401 {string} string "unauthorized"
// @Failure 404 {string} string "task not found"
// @Failure 429 {string} string "rate limit exceeded"
// @Failure 500 {string} string "internal server error"
// @Router /api/v1/tasks/{id} [get]
func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
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

	task, err := h.service.GetTask(r.Context(), userID, taskID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrTaskNotFound),
			errors.Is(err, domain.ErrInsufficientPermissions):
			http.Error(w, "task not found", http.StatusNotFound)

		default:
			logHandlerError(r, "get task failed", err, "task_id", taskID)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}

		return
	}

	resp := taskToResponse(*task)
	if err := writeJSON(w, http.StatusOK, resp); err != nil {
		logHandlerError(r, "write get task response failed", err, "task_id", task.ID)
	}
}

func taskPatchFromRequest(req dto.UpdateTaskRequest) (domain.TaskPatch, error) {
	patch := domain.TaskPatch{
		Version: req.Version,
	}

	if req.Title.Set {
		if req.Title.Value == nil {
			return domain.TaskPatch{}, domain.ErrInvalidTaskTitle
		}

		patch.Title = req.Title.Value
	}

	if req.Description.Set {
		patch.DescriptionSet = true
		patch.Description = req.Description.Value
	}

	if req.Status.Set {
		if req.Status.Value == nil {
			return domain.TaskPatch{}, domain.ErrInvalidTaskStatus
		}

		status := domain.TaskStatus(*req.Status.Value)
		patch.Status = &status
	}

	if req.AssigneeID.Set {
		patch.AssigneeSet = true
		patch.AssigneeID = req.AssigneeID.Value
	}

	return patch, nil
}

// UpdateTask updates task fields using optimistic locking by version.
// @Summary Update task
// @Description Обновление задачи с version (optimistic lock) и записью в историю.
// @Tags tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Task ID"
// @Param request body dto.UpdateTaskDocRequest true "Update task request"
// @Success 200 {object} dto.TaskResponse
// @Failure 400 {string} string "invalid request"
// @Failure 401 {string} string "unauthorized"
// @Failure 403 {string} string "insufficient permissions"
// @Failure 404 {string} string "task not found"
// @Failure 409 {string} string "task version conflict"
// @Failure 429 {string} string "rate limit exceeded"
// @Failure 500 {string} string "internal server error"
// @Router /api/v1/tasks/{id} [put]
func (h *TaskHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
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

	var req dto.UpdateTaskRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	patch, err := taskPatchFromRequest(req)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidTaskTitle),
			errors.Is(err, domain.ErrInvalidTaskStatus):

			http.Error(
				w,
				err.Error(),
				http.StatusBadRequest,
			)

		default:
			logHandlerError(r, "build task patch failed", err, "task_id", taskID)
			http.Error(
				w,
				"internal server error",
				http.StatusInternalServerError,
			)
		}

		return
	}

	task, err := h.service.UpdateTask(
		r.Context(),
		userID,
		taskID,
		patch,
	)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidTaskVersion),
			errors.Is(err, domain.ErrEmptyTaskPatch),
			errors.Is(err, domain.ErrInvalidTaskTitle),
			errors.Is(err, domain.ErrInvalidTaskStatus),
			errors.Is(err, domain.ErrInvalidAssignee),
			errors.Is(err, domain.ErrNoTaskChanges):

			http.Error(
				w,
				err.Error(),
				http.StatusBadRequest,
			)

		case errors.Is(err, domain.ErrInsufficientPermissions):
			http.Error(
				w,
				domain.ErrInsufficientPermissions.Error(),
				http.StatusForbidden,
			)

		case errors.Is(err, domain.ErrTaskNotFound):
			http.Error(
				w,
				domain.ErrTaskNotFound.Error(),
				http.StatusNotFound,
			)

		case errors.Is(err, domain.ErrTaskVersionConflict):
			http.Error(
				w,
				domain.ErrTaskVersionConflict.Error(),
				http.StatusConflict,
			)

		default:
			logHandlerError(r, "update task failed", err, "task_id", taskID)

			http.Error(
				w,
				"internal server error",
				http.StatusInternalServerError,
			)
		}

		return
	}

	resp := taskToResponse(*task)

	if err := writeJSON(
		w,
		http.StatusOK,
		resp,
	); err != nil {
		logHandlerError(r, "write update task response failed", err, "task_id", task.ID)
	}

}
