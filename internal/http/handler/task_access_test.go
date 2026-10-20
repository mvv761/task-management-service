package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"task-management-service/internal/domain"
	appmiddleware "task-management-service/internal/http/middleware"
	"github.com/go-chi/chi/v5"
)

type fakeTaskAccessService struct {
	err error
}

func (f *fakeTaskAccessService) CreateTask(ctx context.Context, teamID uint64, userID uint64, title string, description *string, assigneeID *uint64) (*domain.Task, error) {
	return nil, nil
}

func (f *fakeTaskAccessService) ListTasks(ctx context.Context, userID uint64, filter domain.TaskFilter) ([]domain.Task, error) {
	return nil, nil
}

func (f *fakeTaskAccessService) GetTask(ctx context.Context, userID uint64, taskID uint64) (*domain.Task, error) {
	return nil, f.err
}

func (f *fakeTaskAccessService) UpdateTask(ctx context.Context, userID uint64, taskID uint64, patch domain.TaskPatch) (*domain.Task, error) {
	return nil, nil
}

type fakeTaskHistoryAccessService struct {
	err error
}

func (f *fakeTaskHistoryAccessService) ListTaskHistory(ctx context.Context, userID uint64, taskID uint64) ([]domain.TaskHistory, error) {
	return nil, f.err
}

type fakeTaskCommentAccessService struct {
	err error
}

func (f *fakeTaskCommentAccessService) CreateComment(ctx context.Context, userID uint64, taskID uint64, content string) (*domain.TaskComment, error) {
	return nil, f.err
}

func (f *fakeTaskCommentAccessService) ListComments(ctx context.Context, userID uint64, taskID uint64) ([]domain.TaskComment, error) {
	return nil, f.err
}

func TestTaskHandler_GetTask_MasksPermissionErrorAsNotFound(t *testing.T) {
	const secret = "test-secret"

	router := chi.NewRouter()
	router.Use(appmiddleware.NewAuth(secret).Authenticate)
	router.Get("/api/v1/tasks/{id}", NewTaskHandler(&fakeTaskAccessService{err: domain.ErrInsufficientPermissions}).GetTask)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/10", nil)
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, 99))
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d with body %q", http.StatusNotFound, rr.Code, rr.Body.String())
	}
}

func TestTaskHistoryHandler_ListTaskHistory_MasksPermissionErrorAsNotFound(t *testing.T) {
	const secret = "test-secret"

	router := chi.NewRouter()
	router.Use(appmiddleware.NewAuth(secret).Authenticate)
	router.Get("/api/v1/tasks/{id}/history", NewTaskHistoryHandler(&fakeTaskHistoryAccessService{err: domain.ErrInsufficientPermissions}).ListTaskHistory)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/10/history", nil)
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, 99))
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d with body %q", http.StatusNotFound, rr.Code, rr.Body.String())
	}
}

func TestTaskCommentHandler_CreateComment_MasksPermissionErrorAsNotFound(t *testing.T) {
	const secret = "test-secret"

	router := chi.NewRouter()
	router.Use(appmiddleware.NewAuth(secret).Authenticate)
	router.Post("/api/v1/tasks/{id}/comments", NewTaskCommentHandler(&fakeTaskCommentAccessService{err: domain.ErrInsufficientPermissions}).CreateComment)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/10/comments", strings.NewReader(`{"content":"comment"}`))
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, 99))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d with body %q", http.StatusNotFound, rr.Code, rr.Body.String())
	}
}

func TestTaskCommentHandler_ListComments_MasksPermissionErrorAsNotFound(t *testing.T) {
	const secret = "test-secret"

	router := chi.NewRouter()
	router.Use(appmiddleware.NewAuth(secret).Authenticate)
	router.Get("/api/v1/tasks/{id}/comments", NewTaskCommentHandler(&fakeTaskCommentAccessService{err: domain.ErrInsufficientPermissions}).ListComments)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/10/comments", nil)
	req.Header.Set("Authorization", "Bearer "+testJWT(t, secret, 99))
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d with body %q", http.StatusNotFound, rr.Code, rr.Body.String())
	}
}
