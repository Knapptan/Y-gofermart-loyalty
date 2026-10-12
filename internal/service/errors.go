package service

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrLoginExists        = errors.New("login already exists")
	ErrEmptyCredentials   = errors.New("login and password are required")
)
