package domain

import "errors"

var (
	ErrNotFound      = errors.New("not found")
	ErrInvalidAmount = errors.New("amount must be greater than zero")
	ErrEmptyCategory = errors.New("category cannot be empty")
	ErrEmailExists   = errors.New("email already exists")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrInvalidCreds  = errors.New("invalid email or password")
)
