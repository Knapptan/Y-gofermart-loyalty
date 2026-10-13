package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/knapptan/go-musthave-diploma-tpl/internal/repository"
)

func TestUserCreate_Success(t *testing.T) {
	truncate(t, "withdrawals", "balances", "orders", "users")

	u, err := NewUserRepo(testDB).Create(context.Background(), "alice", "$2b$hash")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if u.ID == "" {
		t.Fatal("empty ID")
	}
	if u.Login != "alice" {
		t.Fatalf("login=%q, want alice", u.Login)
	}
}

func TestUserCreate_DuplicateLogin(t *testing.T) {
	truncate(t, "withdrawals", "balances", "orders", "users")

	repo := NewUserRepo(testDB)
	ctx := context.Background()
	if _, err := repo.Create(ctx, "bob", "$2b$hash"); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	_, err := repo.Create(ctx, "bob", "$2b$another")
	if !errors.Is(err, repository.ErrLoginExists) {
		t.Fatalf("err=%v, want ErrLoginExists", err)
	}
}

func TestUserGetByLogin_Success(t *testing.T) {
	truncate(t, "withdrawals", "balances", "orders", "users")

	repo := NewUserRepo(testDB)
	ctx := context.Background()
	created, _ := repo.Create(ctx, "alice", "$2b$hash")

	got, err := repo.GetByLogin(ctx, "alice")
	if err != nil {
		t.Fatalf("GetByLogin: %v", err)
	}
	if got.ID != created.ID || got.PasswordHash != "$2b$hash" {
		t.Fatalf("got %+v, want id=%s", got, created.ID)
	}
}

func TestUserGetByLogin_NotFound(t *testing.T) {
	truncate(t, "withdrawals", "balances", "orders", "users")

	_, err := NewUserRepo(testDB).GetByLogin(context.Background(), "ghost")
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("err=%v, want ErrNotFound", err)
	}
}
