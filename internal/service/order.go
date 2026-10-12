package service

import (
	"context"
	"errors"

	"github.com/knapptan/go-musthave-diploma-tpl/internal/model"
	"github.com/knapptan/go-musthave-diploma-tpl/internal/repository"
)

// OrderService — сервис управления заказами.
type OrderService struct {
	orders repository.OrderRepository
}

// NewOrderService создаёт OrderService.
func NewOrderService(orders repository.OrderRepository) *OrderService {
	return &OrderService{orders: orders}
}

// CreateOrder валидирует номер по Луну и сохраняет заказ.
func (s *OrderService) CreateOrder(ctx context.Context, userID, number string) error {
	if number == "" || !LuhnValid(number) {
		return ErrInvalidOrderNumber
	}
	return s.orders.Create(ctx, userID, number)
}

// ListOrders возвращает заказы пользователя в порядке убывания uploaded_at.
func (s *OrderService) ListOrders(ctx context.Context, userID string) ([]model.Order, error) {
	return s.orders.ListByUser(ctx, userID)
}

// IsOrderConflict — вспомогательная функция для определения типа конфликта.
func IsOrderConflict(err error) (same bool, other bool) {
	return errors.Is(err, repository.ErrOrderExistsSame),
		errors.Is(err, repository.ErrOrderExistsOther)
}
