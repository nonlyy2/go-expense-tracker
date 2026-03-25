package mock

import (
	"context"
	"sync"

	"go-expense-tracker/internal/domain"
)

// ExpenseRepo is an in-memory implementation of repository.ExpenseRepository for testing.
var _ interface {
	GetAll(ctx context.Context, userID int) ([]domain.Expense, error)
	GetByID(ctx context.Context, id int, userID int) (*domain.Expense, error)
	Create(ctx context.Context, expense *domain.Expense) error
	Update(ctx context.Context, expense *domain.Expense) error
	Delete(ctx context.Context, id int, userID int) error
	GetMonthlyStats(ctx context.Context, userID int) ([]domain.MonthlyStat, error)
	GetCategoryStats(ctx context.Context, userID int) ([]domain.CategoryStat, error)
} = (*ExpenseRepo)(nil)

type ExpenseRepo struct {
	mu       sync.Mutex
	expenses []domain.Expense
	nextID   int

	// injectable error stubs for error-path tests
	CreateErr error
	GetAllErr error
}

func NewExpenseRepo() *ExpenseRepo {
	return &ExpenseRepo{nextID: 1}
}

func (r *ExpenseRepo) Create(ctx context.Context, expense *domain.Expense) error {
	if r.CreateErr != nil {
		return r.CreateErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	expense.ID = r.nextID
	r.nextID++
	cp := *expense
	r.expenses = append(r.expenses, cp)
	return nil
}

func (r *ExpenseRepo) GetAll(ctx context.Context, userID int) ([]domain.Expense, error) {
	if r.GetAllErr != nil {
		return nil, r.GetAllErr
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []domain.Expense
	for _, e := range r.expenses {
		if e.UserID == userID {
			result = append(result, e)
		}
	}
	return result, nil
}

func (r *ExpenseRepo) GetByID(ctx context.Context, id int, userID int) (*domain.Expense, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range r.expenses {
		if e.ID == id && e.UserID == userID {
			cp := e
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *ExpenseRepo) Update(ctx context.Context, expense *domain.Expense) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, e := range r.expenses {
		if e.ID == expense.ID && e.UserID == expense.UserID {
			r.expenses[i] = *expense
			return nil
		}
	}
	return domain.ErrNotFound
}

func (r *ExpenseRepo) Delete(ctx context.Context, id int, userID int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, e := range r.expenses {
		if e.ID == id && e.UserID == userID {
			r.expenses = append(r.expenses[:i], r.expenses[i+1:]...)
			return nil
		}
	}
	return domain.ErrNotFound
}

// GetMonthlyStats returns nil for the in-memory mock (no SQL aggregation).
func (r *ExpenseRepo) GetMonthlyStats(_ context.Context, _ int) ([]domain.MonthlyStat, error) {
	return nil, nil
}

// GetCategoryStats returns nil for the in-memory mock (no SQL aggregation).
func (r *ExpenseRepo) GetCategoryStats(_ context.Context, _ int) ([]domain.CategoryStat, error) {
	return nil, nil
}

// GetMonthlyCategoryStats returns nil for the in-memory mock.
func (r *ExpenseRepo) GetMonthlyCategoryStats(_ context.Context, _ int) ([]domain.MonthCategoryStat, error) {
	return nil, nil
}

// Expenses returns a snapshot of all stored expenses (useful in tests).
func (r *ExpenseRepo) Expenses() []domain.Expense {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := make([]domain.Expense, len(r.expenses))
	copy(cp, r.expenses)
	return cp
}
