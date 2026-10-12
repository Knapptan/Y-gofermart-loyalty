package service

import (
	"context"
	"errors"

	"github.com/knapptan/go-musthave-diploma-tpl/internal/model"
	"github.com/knapptan/go-musthave-diploma-tpl/internal/repository"
)

type OrderService struct {
	orders repository.OrderRepository
}

func NewOrderService(orders repository.OrderRepository) *OrderService {
	return &OrderService{orders: orders}
}

// CreateOrder загружает номер заказа.
// Возвращает ErrInvalidOrderNumber при неверном формате/Луне.
// ErrOrderExistsSame / ErrOrderExistsOther пробрасываются как есть.
func (s *OrderService) CreateOrder(ctx context.Context, userID, number string) error {
	if number == "" || !LuhnValid(number) {
		return ErrInvalidOrderNumber
	}
	return s.orders.Create(ctx, userID, number)
}

func (s *OrderService) ListOrders(ctx context.Context, userID string) ([]model.Order, error) {
	return s.orders.ListByUser(ctx, userID)
}

// вспомогательное — чтобы не тащить errors.Is в handler
func IsOrderConflict(err error) (same bool, other bool) {
	return errors.Is(err, repository.ErrOrderExistsSame),
		errors.Is(err, repository.ErrOrderExistsOther)
}
