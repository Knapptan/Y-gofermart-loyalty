package service

import (
	"context"
	"testing"

	"github.com/knapptan/go-musthave-diploma-tpl/internal/model"
	"github.com/knapptan/go-musthave-diploma-tpl/internal/repository"
)

// fakeUserRepoSimple — минимальная реализация UserRepository для тестов.
type fakeUserRepoSimple struct{}

func (f *fakeUserRepoSimple) Create(_ context.Context, login, hash string) (*model.User, error) {
	return &model.User{ID: "user-1", Login: login, PasswordHash: hash}, nil
}

func (f *fakeUserRepoSimple) GetByLogin(_ context.Context, login string) (*model.User, error) {
	if login == "exists" {
		return &model.User{ID: "user-1", Login: login, PasswordHash: "$2a$10$x"}, nil
	}
	return nil, repository.ErrNotFound
}

func TestNewUserService(t *testing.T) {
	repo := &fakeUserRepoSimple{}
	svc := NewUserService(repo, "secret")
	if svc == nil {
		t.Fatal("expected non-nil UserService")
	}
}

func TestUserService_Register(t *testing.T) {
	svc := NewUserService(&fakeUserRepoSimple{}, "secret")
	token, err := svc.Register(context.Background(), "alice", "pass")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
}

func TestUserService_Register_Empty(t *testing.T) {
	svc := NewUserService(&fakeUserRepoSimple{}, "secret")
	_, err := svc.Register(context.Background(), "", "pass")
	if err != ErrEmptyCredentials {
		t.Fatalf("want ErrEmptyCredentials, got %v", err)
	}
}

func TestUserService_Login_NotFound(t *testing.T) {
	svc := NewUserService(&fakeUserRepoSimple{}, "secret")
	_, err := svc.Login(context.Background(), "ghost", "pass")
	if err != ErrInvalidCredentials {
		t.Fatalf("want ErrInvalidCredentials, got %v", err)
	}
}
