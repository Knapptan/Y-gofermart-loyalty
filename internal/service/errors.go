package service

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrLoginExists        = errors.New("login already exists")
	ErrEmptyCredentials   = errors.New("login and password are required")
	ErrInvalidOrderNumber = errors.New("invalid order number")
	ErrInvalidWithdrawSum = errors.New("invalid withdraw sum")
	ErrInsufficientFunds  = errors.New("insufficient funds")
)
