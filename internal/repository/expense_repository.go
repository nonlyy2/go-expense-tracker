package repository

import (
	"context"
	"go-expense-tracker/internal/domain"
)

type ExpenseRepository interface {
	GetAll(ctx context.Context, userID int) ([]domain.Expense, error)
	GetByID(ctx context.Context, id int, userID int) (*domain.Expense, error)
	Create(ctx context.Context, expense *domain.Expense) error
	Update(ctx context.Context, expense *domain.Expense) error
	Delete(ctx context.Context, id int, userID int) error
}
