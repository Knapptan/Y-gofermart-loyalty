package repository

import (
	"context"
	"errors"

	"github.com/knapptan/go-musthave-diploma-tpl/internal/model"
)

var (
	ErrNotFound          = errors.New("not found")
	ErrLoginExists       = errors.New("login already exists")
	ErrOrderExistsSame   = errors.New("order already uploaded by this user")
	ErrOrderExistsOther  = errors.New("order already uploaded by another user")
	ErrInsufficientFunds = errors.New("insufficient funds")
)

type UserRepository interface {
	Create(ctx context.Context, login, passwordHash string) (*model.User, error)
	GetByLogin(ctx context.Context, login string) (*model.User, error)
}

type OrderRepository interface {
	Create(ctx context.Context, userID, number string) error
	GetByNumber(ctx context.Context, number string) (*model.Order, error)
	ListByUser(ctx context.Context, userID string) ([]model.Order, error)
	ListPending(ctx context.Context) ([]model.Order, error)
	ApplyResult(ctx context.Context, number string, status model.OrderStatus, accrual float64) error
}
type BalanceRepository interface {
	Get(ctx context.Context, userID string) (*model.Balance, error)
	Withdraw(ctx context.Context, userID, order string, sum float64) error
	ListWithdrawals(ctx context.Context, userID string) ([]model.Withdrawal, error)
}
