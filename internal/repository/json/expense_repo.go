package json

import (
	"context"
	"encoding/json"
	"os"
	"sync"

	"go-expense-tracker/internal/domain"
	"go-expense-tracker/internal/repository"
)

var _ repository.ExpenseRepository = (*expenseRepo)(nil)

type expenseRepo struct {
	mu       sync.Mutex
	filePath string
	expenses []domain.Expense
}

func NewExpenseRepo(filePath string) (*expenseRepo, error) {
	// create empty repo
	repo := &expenseRepo{
		filePath: filePath,
		expenses: make([]domain.Expense, 0),
	}

	// read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// no file
			return repo, nil
		}
		return nil, err
	}

	// if file exists, unmarshal to repo.expenses
	err = json.Unmarshal(data, &repo.expenses)
	if err != nil {
		return nil, err
	}

	return repo, nil
}

// saves slice to json
func (r *expenseRepo) saveToFile() error {
	jsonData, err := json.MarshalIndent(r.expenses, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.filePath, jsonData, 0644)
}

func (r *expenseRepo) GetAll(ctx context.Context, userID int) ([]domain.Expense, error) {
	r.mu.Lock()         // mutex
	defer r.mu.Unlock() // guarantee that we unlock mutex after func

	// copy of slice
	result := make([]domain.Expense, len(r.expenses))
	copy(result, r.expenses)

	return result, nil
}

func (r *expenseRepo) GetByID(ctx context.Context, id int, userID int) (*domain.Expense, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, exp := range r.expenses {
		if exp.ID == id {
			return &exp, nil
		}
	}

	return nil, domain.ErrNotFound
}

func (r *expenseRepo) Create(ctx context.Context, expense *domain.Expense) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// generate new ID, maxID+1
	maxID := 0
	for _, e := range r.expenses {
		if e.ID > maxID {
			maxID = e.ID
		}
	}
	expense.ID = maxID + 1

	// add new ID
	r.expenses = append(r.expenses, *expense)

	// save to file
	return r.saveToFile()
}

func (r *expenseRepo) Update(ctx context.Context, expense *domain.Expense) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// search for index of expense to update
	for i, e := range r.expenses {
		if e.ID == expense.ID {
			// update slice
			r.expenses[i] = *expense
			// save
			return r.saveToFile()
		}
	}

	return domain.ErrNotFound
}

func (r *expenseRepo) Delete(ctx context.Context, id int, userID int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	index := -1
	for i, e := range r.expenses {
		if e.ID == id {
			index = i
			break
		}
	}

	if index == -1 {
		return domain.ErrNotFound
	}

	// delete element from slice
	r.expenses = append(r.expenses[:index], r.expenses[index+1:]...)

	// save
	return r.saveToFile()
}
