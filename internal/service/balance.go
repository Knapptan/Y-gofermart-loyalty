package service

import (
	"context"
	"errors"

	"github.com/knapptan/go-musthave-diploma-tpl/internal/model"
	"github.com/knapptan/go-musthave-diploma-tpl/internal/repository"
)

// BalanceService — сервис управления балансом и списаниями.
type BalanceService struct {
	balances repository.BalanceRepository
}

// NewBalanceService создаёт BalanceService.
func NewBalanceService(balances repository.BalanceRepository) *BalanceService {
	return &BalanceService{balances: balances}
}

// Get возвращает баланс пользователя по ID.
func (s *BalanceService) Get(ctx context.Context, userID string) (*model.Balance, error) {
	return s.balances.Get(ctx, userID)
}

// Withdraw списывает баллы: валидирует номер (Лун), проверяет сумму, списывает транзакцией.
func (s *BalanceService) Withdraw(ctx context.Context, userID, order string, sum float64) error {
	if !LuhnValid(order) {
		return ErrInvalidOrderNumber
	}
	if sum <= 0 {
		return ErrInvalidWithdrawSum
	}

	err := s.balances.Withdraw(ctx, userID, order, sum)
	if errors.Is(err, repository.ErrInsufficientFunds) {
		return ErrInsufficientFunds
	}
	return err
}

// ListWithdrawals возвращает списания пользователя (DESC по processed_at).
func (s *BalanceService) ListWithdrawals(ctx context.Context, userID string) ([]model.Withdrawal, error) {
	return s.balances.ListWithdrawals(ctx, userID)
}
