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
	GetMonthlyStats(ctx context.Context, userID int) ([]domain.MonthlyStat, error)
	GetCategoryStats(ctx context.Context, userID int) ([]domain.CategoryStat, error)
	GetMonthlyCategoryStats(ctx context.Context, userID int) ([]domain.MonthCategoryStat, error)
}
