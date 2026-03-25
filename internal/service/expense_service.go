package service

import (
	"context"
	"time"

	"go-expense-tracker/internal/domain"
	"go-expense-tracker/internal/repository"
)

type ExpenseService struct {
	repo repository.ExpenseRepository
}

func NewExpenseService(repo repository.ExpenseRepository) *ExpenseService {
	return &ExpenseService{
		repo: repo,
	}
}

// CreateExpense checks data and creates new expense
func (s *ExpenseService) CreateExpense(ctx context.Context, category string, amount float64, comment string, userID int) (*domain.Expense, error) {
	// validation
	if amount <= 0 {
		return nil, domain.ErrInvalidAmount
	}
	if category == "" {
		return nil, domain.ErrEmptyCategory
	}

	// new struct
	exp := domain.Expense{
		Date:     time.Now(),
		Amount:   amount,
		Category: category,
		Comment:  comment,
		UserID:   userID,
	}

	// save to repo
	if err := s.repo.Create(ctx, &exp); err != nil {
		return nil, err
	}

	return &exp, nil
}

// GetAllExpenses requests data from repo
func (s *ExpenseService) GetAllExpenses(ctx context.Context, userID int) ([]domain.Expense, error) {
	return s.repo.GetAll(ctx, userID)
}

func (s *ExpenseService) GetExpenseByID(ctx context.Context, id int, userID int) (*domain.Expense, error) {
	return s.repo.GetByID(ctx, id, userID)
}

func (s *ExpenseService) UpdateExpense(ctx context.Context, id int, category string, amount float64, comment string, userID int) (*domain.Expense, error) {
	// validation
	if amount <= 0 {
		return nil, domain.ErrInvalidAmount
	}
	if category == "" {
		return nil, domain.ErrEmptyCategory
	}

	// expense exists?
	existingExp, err := s.repo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	existingExp.Category = category
	existingExp.Amount = amount
	existingExp.Comment = comment

	// save to repo
	if err := s.repo.Update(ctx, existingExp); err != nil {
		return nil, err
	}

	return existingExp, nil
}

func (s *ExpenseService) DeleteExpense(ctx context.Context, id int, userID int) error {
	return s.repo.Delete(ctx, id, userID)
}

func (s *ExpenseService) GetMonthlyStats(ctx context.Context, userID int) ([]domain.MonthlyStat, error) {
	return s.repo.GetMonthlyStats(ctx, userID)
}

func (s *ExpenseService) GetCategoryStats(ctx context.Context, userID int) ([]domain.CategoryStat, error) {
	return s.repo.GetCategoryStats(ctx, userID)
}

func (s *ExpenseService) GetMonthlyCategoryStats(ctx context.Context, userID int) ([]domain.MonthCategoryStat, error) {
	return s.repo.GetMonthlyCategoryStats(ctx, userID)
}

// calc total amount of expenses
func (s *ExpenseService) GetTotal(ctx context.Context, userID int) (float64, error) {
	expenses, err := s.repo.GetAll(ctx, userID)
	if err != nil {
		return 0, err
	}

	var total float64
	for _, e := range expenses {
		total += e.Amount
	}

	return total, nil
}
