package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"task-management-service/internal/domain"
	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

type UserRepository struct {
	db *sqlx.DB
}

const mysqlDuplicateEntryError uint16 = 1062

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(
	ctx context.Context,
	user *domain.User,
) error {
	query := `
		INSERT INTO users (email, password_hash, name)
		VALUES (?, ?, ?)
	`

	res, err := r.db.ExecContext(
		ctx,
		query,
		user.Email,
		user.PasswordHash,
		user.Name,
	)
	if err != nil {
		if mysqlErr, ok := errors.AsType[*mysqldriver.MySQLError](err); ok &&
			mysqlErr.Number == mysqlDuplicateEntryError {
			return domain.ErrEmailAlreadyTaken
		}

		return fmt.Errorf("create user: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("get user id: %w", err)
	}

	user.ID = uint64(id)

	return nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	query := "SELECT id, email, password_hash, name, created_at FROM users WHERE email = ?"
	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id uint64) (*domain.User, error) {
	var user domain.User
	query := "SELECT id, email, password_hash, name, created_at FROM users WHERE id = ?"
	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	return &user, nil
}
