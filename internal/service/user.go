package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/knapptan/go-musthave-diploma-tpl/internal/auth"
	"github.com/knapptan/go-musthave-diploma-tpl/internal/repository"
)

const tokenTTL = 24 * time.Hour

type UserService struct {
	users     repository.UserRepository
	jwtSecret string
}

func NewUserService(users repository.UserRepository, jwtSecret string) *UserService {
	return &UserService{users: users, jwtSecret: jwtSecret}
}

// Register создаёт пользователя и возвращает JWT-токен.
func (s *UserService) Register(ctx context.Context, login, password string) (string, error) {
	login = strings.TrimSpace(login)
	if login == "" || password == "" {
		return "", ErrEmptyCredentials
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return "", err
	}

	u, err := s.users.Create(ctx, login, hash)
	if err != nil {
		if errors.Is(err, repository.ErrLoginExists) {
			return "", ErrLoginExists
		}
		return "", err
	}

	return auth.NewToken(s.jwtSecret, u.ID, tokenTTL)
}

// Login проверяет пару логин/пароль и возвращает JWT-токен.
func (s *UserService) Login(ctx context.Context, login, password string) (string, error) {
	login = strings.TrimSpace(login)
	if login == "" || password == "" {
		return "", ErrEmptyCredentials
	}

	u, err := s.users.GetByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", ErrInvalidCredentials
		}
		return "", err
	}

	if !auth.CheckPassword(u.PasswordHash, password) {
		return "", ErrInvalidCredentials
	}

	return auth.NewToken(s.jwtSecret, u.ID, tokenTTL)
}
