package service

import (
	"context"
	"errors"
	"testing"

	"go-expense-tracker/internal/domain"
	"go-expense-tracker/internal/repository/mock"
)

// ─── helpers ────────────────────────────────────────────────────────────────

func newSvc() (*ExpenseService, *mock.ExpenseRepo) {
	repo := mock.NewExpenseRepo()
	return NewExpenseService(repo), repo
}

// ─── CreateExpense ───────────────────────────────────────────────────────────

func TestCreateExpense_Valid(t *testing.T) {
	svc, _ := newSvc()
	exp, err := svc.CreateExpense(context.Background(), "Food", 500, "lunch", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exp.ID == 0 {
		t.Error("expected non-zero ID after create")
	}
	if exp.Category != "Food" || exp.Amount != 500 {
		t.Errorf("unexpected values: %+v", exp)
	}
}

func TestCreateExpense_ZeroAmount(t *testing.T) {
	svc, _ := newSvc()
	_, err := svc.CreateExpense(context.Background(), "Food", 0, "", 1)
	if !errors.Is(err, domain.ErrInvalidAmount) {
		t.Errorf("got %v, want ErrInvalidAmount", err)
	}
}

func TestCreateExpense_NegativeAmount(t *testing.T) {
	svc, _ := newSvc()
	_, err := svc.CreateExpense(context.Background(), "Food", -100, "", 1)
	if !errors.Is(err, domain.ErrInvalidAmount) {
		t.Errorf("got %v, want ErrInvalidAmount", err)
	}
}

func TestCreateExpense_EmptyCategory(t *testing.T) {
	svc, _ := newSvc()
	_, err := svc.CreateExpense(context.Background(), "", 100, "", 1)
	if !errors.Is(err, domain.ErrEmptyCategory) {
		t.Errorf("got %v, want ErrEmptyCategory", err)
	}
}

func TestCreateExpense_RepoError(t *testing.T) {
	svc, repo := newSvc()
	repo.CreateErr = errors.New("db unavailable")
	_, err := svc.CreateExpense(context.Background(), "Food", 100, "", 1)
	if err == nil || err.Error() != "db unavailable" {
		t.Errorf("expected repo error, got %v", err)
	}
}

// table-driven validation cases
func TestCreateExpense_Table(t *testing.T) {
	cases := []struct {
		name     string
		category string
		amount   float64
		wantErr  error
	}{
		{"valid", "Transport", 250, nil},
		{"zero amount", "Transport", 0, domain.ErrInvalidAmount},
		{"negative amount", "Transport", -1, domain.ErrInvalidAmount},
		{"empty category", "", 250, domain.ErrEmptyCategory},
		{"very small amount", "Other", 0.01, nil},
		{"large amount", "Shopping", 999999.99, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, _ := newSvc()
			_, err := svc.CreateExpense(context.Background(), tc.category, tc.amount, "test", 1)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("got %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// ─── GetAllExpenses ──────────────────────────────────────────────────────────

func TestGetAllExpenses_Empty(t *testing.T) {
	svc, _ := newSvc()
	expenses, err := svc.GetAllExpenses(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(expenses) != 0 {
		t.Errorf("expected 0 expenses, got %d", len(expenses))
	}
}

func TestGetAllExpenses_IsolationByUser(t *testing.T) {
	svc, _ := newSvc()
	ctx := context.Background()

	svc.CreateExpense(ctx, "Food", 100, "", 1)
	svc.CreateExpense(ctx, "Food", 200, "", 1)
	svc.CreateExpense(ctx, "Food", 300, "", 2) // different user

	expenses, err := svc.GetAllExpenses(ctx, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(expenses) != 2 {
		t.Errorf("got %d expenses for user 1, want 2", len(expenses))
	}
	for _, e := range expenses {
		if e.UserID != 1 {
			t.Errorf("expense belongs to user %d, not 1", e.UserID)
		}
	}
}

func TestGetAllExpenses_RepoError(t *testing.T) {
	svc, repo := newSvc()
	repo.GetAllErr = errors.New("db unavailable")
	_, err := svc.GetAllExpenses(context.Background(), 1)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// ─── GetExpenseByID ──────────────────────────────────────────────────────────

func TestGetExpenseByID_Found(t *testing.T) {
	svc, _ := newSvc()
	exp, _ := svc.CreateExpense(context.Background(), "Food", 500, "test", 1)

	got, err := svc.GetExpenseByID(context.Background(), exp.ID, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != exp.ID {
		t.Errorf("got ID %d, want %d", got.ID, exp.ID)
	}
}

func TestGetExpenseByID_NotFound(t *testing.T) {
	svc, _ := newSvc()
	_, err := svc.GetExpenseByID(context.Background(), 999, 1)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

func TestGetExpenseByID_WrongUser(t *testing.T) {
	svc, _ := newSvc()
	exp, _ := svc.CreateExpense(context.Background(), "Food", 500, "", 1)

	// user 2 tries to access user 1's expense
	_, err := svc.GetExpenseByID(context.Background(), exp.ID, 2)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound (cross-user access should be denied)", err)
	}
}

// ─── UpdateExpense ───────────────────────────────────────────────────────────

func TestUpdateExpense_Valid(t *testing.T) {
	svc, _ := newSvc()
	exp, _ := svc.CreateExpense(context.Background(), "Food", 500, "original", 1)

	updated, err := svc.UpdateExpense(context.Background(), exp.ID, "Transport", 750, "updated", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Category != "Transport" || updated.Amount != 750 || updated.Comment != "updated" {
		t.Errorf("unexpected updated values: %+v", updated)
	}
}

func TestUpdateExpense_NotFound(t *testing.T) {
	svc, _ := newSvc()
	_, err := svc.UpdateExpense(context.Background(), 999, "Food", 100, "", 1)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

func TestUpdateExpense_InvalidAmount(t *testing.T) {
	svc, _ := newSvc()
	exp, _ := svc.CreateExpense(context.Background(), "Food", 500, "", 1)
	_, err := svc.UpdateExpense(context.Background(), exp.ID, "Food", 0, "", 1)
	if !errors.Is(err, domain.ErrInvalidAmount) {
		t.Errorf("got %v, want ErrInvalidAmount", err)
	}
}

func TestUpdateExpense_EmptyCategory(t *testing.T) {
	svc, _ := newSvc()
	exp, _ := svc.CreateExpense(context.Background(), "Food", 500, "", 1)
	_, err := svc.UpdateExpense(context.Background(), exp.ID, "", 500, "", 1)
	if !errors.Is(err, domain.ErrEmptyCategory) {
		t.Errorf("got %v, want ErrEmptyCategory", err)
	}
}

// ─── DeleteExpense ───────────────────────────────────────────────────────────

func TestDeleteExpense_Valid(t *testing.T) {
	svc, _ := newSvc()
	exp, _ := svc.CreateExpense(context.Background(), "Food", 500, "", 1)

	if err := svc.DeleteExpense(context.Background(), exp.ID, 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// verify gone
	_, err := svc.GetExpenseByID(context.Background(), exp.ID, 1)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Error("expense should be deleted but was found")
	}
}

func TestDeleteExpense_NotFound(t *testing.T) {
	svc, _ := newSvc()
	err := svc.DeleteExpense(context.Background(), 999, 1)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

func TestDeleteExpense_WrongUser(t *testing.T) {
	svc, _ := newSvc()
	exp, _ := svc.CreateExpense(context.Background(), "Food", 500, "", 1)

	// user 2 tries to delete user 1's expense
	err := svc.DeleteExpense(context.Background(), exp.ID, 2)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound (cross-user delete should be denied)", err)
	}
	// original should still exist
	_, err = svc.GetExpenseByID(context.Background(), exp.ID, 1)
	if err != nil {
		t.Errorf("expense should still exist after failed cross-user delete: %v", err)
	}
}

// ─── GetTotal ────────────────────────────────────────────────────────────────

func TestGetTotal_Empty(t *testing.T) {
	svc, _ := newSvc()
	total, err := svc.GetTotal(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 0 {
		t.Errorf("got %.2f, want 0", total)
	}
}

func TestGetTotal_Correct(t *testing.T) {
	svc, _ := newSvc()
	ctx := context.Background()
	svc.CreateExpense(ctx, "Food", 100, "", 1)
	svc.CreateExpense(ctx, "Transport", 250.50, "", 1)
	svc.CreateExpense(ctx, "Other", 50, "", 1)

	total, err := svc.GetTotal(ctx, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 400.50 {
		t.Errorf("got %.2f, want 400.50", total)
	}
}

func TestGetTotal_IsolationByUser(t *testing.T) {
	svc, _ := newSvc()
	ctx := context.Background()
	svc.CreateExpense(ctx, "Food", 500, "", 1)
	svc.CreateExpense(ctx, "Food", 1000, "", 2)

	total, _ := svc.GetTotal(ctx, 1)
	if total != 500 {
		t.Errorf("got %.2f, want 500 (user 2 expenses should not be counted)", total)
	}
}

// ─── Concurrency ─────────────────────────────────────────────────────────────

func TestCreateExpense_Concurrent(t *testing.T) {
	svc, _ := newSvc()
	ctx := context.Background()
	done := make(chan struct{}, 50)

	for i := 0; i < 50; i++ {
		go func() {
			svc.CreateExpense(ctx, "Food", 100, "", 1)
			done <- struct{}{}
		}()
	}
	for i := 0; i < 50; i++ {
		<-done
	}

	expenses, _ := svc.GetAllExpenses(ctx, 1)
	if len(expenses) != 50 {
		t.Errorf("got %d expenses after 50 concurrent creates, want 50", len(expenses))
	}
}
