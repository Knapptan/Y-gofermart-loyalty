// Package postgres — PostgreSQL-реализации интерфейсов repository.
package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/knapptan/go-musthave-diploma-tpl/internal/model"
	"github.com/knapptan/go-musthave-diploma-tpl/internal/repository"
)

// UserRepo — PostgreSQL-реализация UserRepository.
type UserRepo struct {
	db *sql.DB
}

// NewUserRepo создаёт UserRepo.
func NewUserRepo(db *sql.DB) *UserRepo { return &UserRepo{db: db} }

// Create вставляет пользователя; маппит UNIQUE violation → ErrLoginExists.
func (r *UserRepo) Create(ctx context.Context, login, passwordHash string) (*model.User, error) {
	id := uuid.NewString()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, login, password_hash) VALUES ($1, $2, $3)`,
		id, login, passwordHash,
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, repository.ErrLoginExists
		}
		return nil, err
	}
	return &model.User{ID: id, Login: login, PasswordHash: passwordHash}, nil
}

// GetByLogin ищет пользователя по login.
func (r *UserRepo) GetByLogin(ctx context.Context, login string) (*model.User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, login, password_hash FROM users WHERE login = $1`, login)
	var u model.User
	if err := row.Scan(&u.ID, &u.Login, &u.PasswordHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}
