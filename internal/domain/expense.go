package domain

import (
	"time"
)

// main business entity
type Expense struct {
	ID       int       `json:"id"`
	UserID   int       `json:"user_id"`
	Date     time.Time `json:"date"`
	Amount   float64   `json:"amount"`
	Category string    `json:"category"`
	Comment  string    `json:"comment"`
}
