package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"task-management-service/internal/domain"
	"task-management-service/internal/http/dto"
)

type AuthService interface {
	Register(
		ctx context.Context,
		email string,
		password string,
		name string,
	) (*domain.User, error)

	Login(
		ctx context.Context,
		email string,
		password string,
	) (string, error)
}

type AuthHandler struct {
	service AuthService
}

func NewAuthHandler(service AuthService) *AuthHandler {
	return &AuthHandler{
		service: service,
	}
}

// Register creates a new user account.
// @Summary Register
// @Description Регистрация пользователя.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.RegisterRequest true "Register request"
// @Success 201 {object} dto.RegisterResponse
// @Failure 400 {string} string "invalid request"
// @Failure 409 {string} string "email already taken"
// @Failure 429 {string} string "rate limit exceeded"
// @Failure 500 {string} string "internal server error"
// @Router /api/v1/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	user, err := h.service.Register(r.Context(), req.Email, req.Password, req.Name)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidEmail),
			errors.Is(err, domain.ErrInvalidName),
			errors.Is(err, domain.ErrPasswordTooShort):
			http.Error(w, err.Error(), http.StatusBadRequest)

		case errors.Is(err, domain.ErrEmailAlreadyTaken):
			http.Error(w, err.Error(), http.StatusConflict)

		default:
			logHandlerError(r, "register user failed", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}

		return
	}

	resp := dto.RegisterResponse{
		ID:    user.ID,
		Email: user.Email,
		Name:  user.Name,
	}

	if err := writeJSON(w, http.StatusCreated, resp); err != nil {
		logHandlerError(r, "write register response failed", err)
	}
}

// Login authenticates a user and returns a JWT access token.
// @Summary Login
// @Description Логин, в ответе JWT для последующих запросов.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "Login request"
// @Success 200 {object} dto.LoginResponse
// @Failure 400 {string} string "invalid request"
// @Failure 401 {string} string "invalid credentials"
// @Failure 429 {string} string "rate limit exceeded"
// @Failure 500 {string} string "internal server error"
// @Router /api/v1/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	token, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			http.Error(
				w,
				domain.ErrInvalidCredentials.Error(),
				http.StatusUnauthorized,
			)
			return
		}

		logHandlerError(r, "login user failed", err)
		http.Error(
			w,
			"internal server error",
			http.StatusInternalServerError,
		)
		return
	}

	resp := dto.LoginResponse{
		AccessToken: token,
	}

	if err := writeJSON(w, http.StatusOK, resp); err != nil {
		logHandlerError(r, "write login response failed", err)
	}
}
