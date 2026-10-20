package service

import (
	"context"
	"testing"
	"time"

	"task-management-service/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type fakeUserRepository struct {
	findByEmailFunc func(
		ctx context.Context,
		email string,
	) (*domain.User, error)
}

func (f *fakeUserRepository) Create(
	ctx context.Context,
	user *domain.User,
) error {
	return nil
}

func (f *fakeUserRepository) FindByEmail(
	ctx context.Context,
	email string,
) (*domain.User, error) {
	return f.findByEmailFunc(ctx, email)
}
func TestAuthService_Login_Success(t *testing.T) {
	password := "testpass"

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}

	repo := &fakeUserRepository{
		findByEmailFunc: func(
			ctx context.Context,
			email string,
		) (*domain.User, error) {
			return &domain.User{
				ID:           1,
				Email:        email,
				PasswordHash: string(passwordHash),
				Name:         "TestUser",
			}, nil
		},
	}

	authService := NewAuthService(
		repo,
		"test-secret",
		time.Hour,
	)

	_, err = authService.Login(
		context.Background(),
		"test@example.com",
		password,
	)
	if err != nil {
		t.Fatal(err)
	}
}
