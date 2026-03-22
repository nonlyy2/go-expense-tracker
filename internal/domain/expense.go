package domain

import (
	"errors"
	"time"
)

// custom errors
var (
	ErrNotFound      = errors.New("expense not found")
	ErrInvalidAmount = errors.New("amount must be greater than zero")
	ErrEmptyCategory = errors.New("category cannot be empty")
)

// main business entity
type Expense struct {
	ID       int       `json:"id"`
	Date     time.Time `json:"date"`
	Amount   float64   `json:"amount"`
	Category string    `json:"category"`
	Comment  string    `json:"comment"`
}
