// Package model содержит доменные сущности сервиса лояльности:
// User, Order, Balance, Withdrawal и статусы заказов.
package model

import "time"

// OrderStatus — статус обработки заказа.
type OrderStatus string

const (
	// StatusNew — заказ загружен, но не попал в обработку.
	StatusNew OrderStatus = "NEW"
	// StatusProcessing — вознаграждение рассчитывается.
	StatusProcessing OrderStatus = "PROCESSING"
	// StatusInvalid — отказ в расчёте.
	StatusInvalid OrderStatus = "INVALID"
	// StatusProcessed — расчёт успешно завершён.
	StatusProcessed OrderStatus = "PROCESSED"
)

// User — зарегистрированный пользователь.
type User struct {
	ID           string
	Login        string
	PasswordHash string
}

// Order — номер заказа, загруженный пользователем.
type Order struct {
	Number     string      `json:"number"`
	UserID     string      `json:"-"`
	Status     OrderStatus `json:"status"`
	Accrual    float64     `json:"accrual,omitempty"`
	UploadedAt time.Time   `json:"uploaded_at"`
}

// Balance — баланс баллов пользователя.
type Balance struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

// Withdrawal — факт списания баллов.
type Withdrawal struct {
	Order       string    `json:"order"`
	Sum         float64   `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
}
