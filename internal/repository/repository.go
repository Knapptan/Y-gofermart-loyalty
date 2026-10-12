// Package repository описывает интерфейсы хранилищ и sentinel-ошибки,
// которые они возвращают.
package repository

import (
	"context"
	"errors"

	"github.com/knapptan/go-musthave-diploma-tpl/internal/model"
)

var (
	// ErrNotFound — сущность не найдена.
	ErrNotFound = errors.New("not found")
	// ErrLoginExists — такой логин уже зарегистрирован.
	ErrLoginExists = errors.New("login already exists")
	// ErrOrderExistsSame — заказ уже загружен этим же пользователем.
	ErrOrderExistsSame = errors.New("order already uploaded by this user")
	// ErrOrderExistsOther — заказ уже загружен другим пользователем.
	ErrOrderExistsOther = errors.New("order already uploaded by another user")
	// ErrInsufficientFunds — на балансе недостаточно средств.
	ErrInsufficientFunds = errors.New("insufficient funds")
)

// UserRepository — интерфейс работы с пользователями.
type UserRepository interface {
	Create(ctx context.Context, login, passwordHash string) (*model.User, error)
	GetByLogin(ctx context.Context, login string) (*model.User, error)
}

// OrderRepository — интерфейс работы с заказами.
type OrderRepository interface {
	Create(ctx context.Context, userID, number string) error
	GetByNumber(ctx context.Context, number string) (*model.Order, error)
	ListByUser(ctx context.Context, userID string) ([]model.Order, error)
	ListPending(ctx context.Context) ([]model.Order, error)
	ApplyResult(ctx context.Context, number string, status model.OrderStatus, accrual float64) error
}

// BalanceRepository — интерфейс работы с балансом и списаниями.
type BalanceRepository interface {
	Get(ctx context.Context, userID string) (*model.Balance, error)
	Withdraw(ctx context.Context, userID, order string, sum float64) error
	ListWithdrawals(ctx context.Context, userID string) ([]model.Withdrawal, error)
}
