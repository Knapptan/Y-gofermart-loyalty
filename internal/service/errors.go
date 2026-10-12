package service

import "errors"

var (
	// ErrInvalidCredentials — неверная пара логин/пароль.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrLoginExists — логин уже занят.
	ErrLoginExists = errors.New("login already exists")
	// ErrEmptyCredentials — логин или пароль пустые.
	ErrEmptyCredentials = errors.New("login and password are required")
	// ErrInvalidOrderNumber — номер заказа не проходит валидацию по Луну.
	ErrInvalidOrderNumber = errors.New("invalid order number")
	// ErrInvalidWithdrawSum — некорректная сумма списания.
	ErrInvalidWithdrawSum = errors.New("invalid withdraw sum")
	// ErrInsufficientFunds — недостаточно средств на балансе.
	ErrInsufficientFunds = errors.New("insufficient funds")
)
