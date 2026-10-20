package domain

import (
	"errors"
	"time"
)

type User struct {
	ID           uint64    `db:"id"`
	Email        string    `db:"email"`
	PasswordHash string    `db:"password_hash"`
	Name         string    `db:"name"`
	CreatedAt    time.Time `db:"created_at"`
}

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailAlreadyTaken  = errors.New("email already taken")
	ErrPasswordTooShort   = errors.New("password too short")
	ErrInvalidName        = errors.New("invalid name")
	ErrInvalidEmail       = errors.New("invalid email")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidTeamRole    = errors.New("invalid team role")
)
