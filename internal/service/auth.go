package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"task-management-service/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
}

type AuthService struct {
	userRepository UserRepository
	jwtSecret      []byte
	jwtTTL         time.Duration
}

func NewAuthService(userRepository UserRepository, jwtSecret string, jwtTTL time.Duration) *AuthService {
	return &AuthService{
		userRepository: userRepository,
		jwtSecret:      []byte(jwtSecret),
		jwtTTL:         jwtTTL,
	}
}

func (s *AuthService) Register(ctx context.Context, email string, password string, name string) (*domain.User, error) {
	email = strings.TrimSpace(email)
	name = strings.TrimSpace(name)

	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return nil, domain.ErrInvalidEmail
	}

	if utf8.RuneCountInString(password) < 8 {
		return nil, domain.ErrPasswordTooShort
	}

	if name == "" {
		return nil, domain.ErrInvalidName
	}

	_, err = s.userRepository.FindByEmail(ctx, email)

	if err == nil {
		return nil, domain.ErrEmailAlreadyTaken
	}

	if !errors.Is(err, domain.ErrUserNotFound) {
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("generate password hash: %w", err)
	}

	user := domain.User{
		Email:        email,
		PasswordHash: string(passwordHash),
		Name:         name,
	}

	if err := s.userRepository.Create(ctx, &user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return &user, nil
}

func (s *AuthService) Login(
	ctx context.Context,
	email string,
	password string,
) (string, error) {
	email = strings.TrimSpace(email)

	user, err := s.userRepository.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return "", domain.ErrInvalidCredentials
		}

		return "", fmt.Errorf("find user by email: %w", err)
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(password),
	)
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return "", domain.ErrInvalidCredentials
		}

		return "", fmt.Errorf("compare password hash: %w", err)
	}

	now := time.Now()

	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatUint(user.ID, 10),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.jwtTTL)),
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("sign jwt token: %w", err)
	}

	return tokenString, nil
}
