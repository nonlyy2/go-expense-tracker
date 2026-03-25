package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-expense-tracker/internal/domain"
	"go-expense-tracker/internal/middleware"
	"go-expense-tracker/internal/repository/mock"
	"go-expense-tracker/internal/service"
)

// ─── helpers ─────────────────────────────────────────────────────────────────

// withUserID injects a userID into the request context (replaces auth middleware in tests).
func withUserID(r *http.Request, userID int) *http.Request {
	ctx := context.WithValue(r.Context(), middleware.UserIDKey, userID)
	return r.WithContext(ctx)
}

func setupHandler() (*ExpenseHandler, *mock.ExpenseRepo) {
	repo := mock.NewExpenseRepo()
	svc := service.NewExpenseService(repo)
	return NewExpenseHandler(svc), repo
}

func decodeJSON(t *testing.T, body *bytes.Buffer, v any) {
	t.Helper()
	if err := json.NewDecoder(body).Decode(v); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
}

// ─── CreateExpense ────────────────────────────────────────────────────────────

func TestCreateExpenseHandler_Created(t *testing.T) {
	h, _ := setupHandler()
	body := `{"category":"Food","amount":500,"comment":"lunch"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/expenses", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUserID(req, 1)

	rr := httptest.NewRecorder()
	h.CreateExpense(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("got %d, want %d", rr.Code, http.StatusCreated)
	}
	var exp domain.Expense
	decodeJSON(t, rr.Body, &exp)
	if exp.ID == 0 {
		t.Error("expected non-zero ID in response")
	}
	if exp.Category != "Food" || exp.Amount != 500 {
		t.Errorf("unexpected response values: %+v", exp)
	}
}

func TestCreateExpenseHandler_InvalidAmount(t *testing.T) {
	h, _ := setupHandler()
	body := `{"category":"Food","amount":-100}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/expenses", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUserID(req, 1)

	rr := httptest.NewRecorder()
	h.CreateExpense(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("got %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestCreateExpenseHandler_ZeroAmount(t *testing.T) {
	h, _ := setupHandler()
	body := `{"category":"Food","amount":0}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/expenses", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUserID(req, 1)

	rr := httptest.NewRecorder()
	h.CreateExpense(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400 for zero amount", rr.Code)
	}
}

func TestCreateExpenseHandler_EmptyCategory(t *testing.T) {
	h, _ := setupHandler()
	body := `{"category":"","amount":100}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/expenses", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUserID(req, 1)

	rr := httptest.NewRecorder()
	h.CreateExpense(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400 for empty category", rr.Code)
	}
}

func TestCreateExpenseHandler_MalformedJSON(t *testing.T) {
	h, _ := setupHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/expenses", bytes.NewBufferString(`not-json`))
	req.Header.Set("Content-Type", "application/json")
	req = withUserID(req, 1)

	rr := httptest.NewRecorder()
	h.CreateExpense(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400 for malformed JSON", rr.Code)
	}
}

func TestCreateExpenseHandler_NoUserID(t *testing.T) {
	h, _ := setupHandler()
	body := `{"category":"Food","amount":100}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/expenses", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	// intentionally no withUserID

	rr := httptest.NewRecorder()
	h.CreateExpense(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("got %d, want 401 when userID missing from context", rr.Code)
	}
}

func TestCreateExpenseHandler_RepoError(t *testing.T) {
	h, repo := setupHandler()
	repo.CreateErr = errors.New("db unavailable")

	body := `{"category":"Food","amount":100}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/expenses", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUserID(req, 1)

	rr := httptest.NewRecorder()
	h.CreateExpense(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("got %d, want 500 on repo error", rr.Code)
	}
}

// ─── GetAllExpenses ───────────────────────────────────────────────────────────

func TestGetAllExpensesHandler_Empty(t *testing.T) {
	h, _ := setupHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/expenses", nil)
	req = withUserID(req, 1)

	rr := httptest.NewRecorder()
	h.GetAllExpenses(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("got %d, want 200", rr.Code)
	}
	// empty list — body should be null or []
	body := rr.Body.String()
	if body != "null\n" && body != "[]\n" {
		// both are acceptable for an empty slice
		var arr []domain.Expense
		if err := json.Unmarshal([]byte(body), &arr); err == nil && len(arr) == 0 {
			return
		}
		if body != "null\n" {
			t.Errorf("unexpected body for empty list: %q", body)
		}
	}
}

func TestGetAllExpensesHandler_WithData(t *testing.T) {
	h, repo := setupHandler()
	ctx := context.Background()
	// seed two expenses for user 1 directly via repo
	repo.Create(ctx, &domain.Expense{UserID: 1, Category: "Food", Amount: 100})
	repo.Create(ctx, &domain.Expense{UserID: 1, Category: "Transport", Amount: 200})
	repo.Create(ctx, &domain.Expense{UserID: 2, Category: "Other", Amount: 50}) // different user

	req := httptest.NewRequest(http.MethodGet, "/api/v1/expenses", nil)
	req = withUserID(req, 1)

	rr := httptest.NewRecorder()
	h.GetAllExpenses(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("got %d, want 200", rr.Code)
	}
	var expenses []domain.Expense
	decodeJSON(t, rr.Body, &expenses)
	if len(expenses) != 2 {
		t.Errorf("got %d expenses, want 2", len(expenses))
	}
}

func TestGetAllExpensesHandler_NoUserID(t *testing.T) {
	h, _ := setupHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/expenses", nil)

	rr := httptest.NewRecorder()
	h.GetAllExpenses(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("got %d, want 401", rr.Code)
	}
}

// ─── GetExpenseByID ───────────────────────────────────────────────────────────

func TestGetExpenseByIDHandler_Found(t *testing.T) {
	h, repo := setupHandler()
	exp := &domain.Expense{UserID: 1, Category: "Food", Amount: 300}
	repo.Create(context.Background(), exp)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/expenses/{id}", h.GetExpenseByID)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/expenses/1", nil)
	req = withUserID(req, 1)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("got %d, want 200", rr.Code)
	}
}

func TestGetExpenseByIDHandler_NotFound(t *testing.T) {
	h, _ := setupHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/expenses/{id}", h.GetExpenseByID)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/expenses/999", nil)
	req = withUserID(req, 1)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("got %d, want 404", rr.Code)
	}
}

func TestGetExpenseByIDHandler_InvalidID(t *testing.T) {
	h, _ := setupHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/expenses/{id}", h.GetExpenseByID)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/expenses/abc", nil)
	req = withUserID(req, 1)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("got %d, want 400", rr.Code)
	}
}

// ─── DeleteExpense ────────────────────────────────────────────────────────────

func TestDeleteExpenseHandler_OK(t *testing.T) {
	h, repo := setupHandler()
	exp := &domain.Expense{UserID: 1, Category: "Food", Amount: 100}
	repo.Create(context.Background(), exp)

	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v1/expenses/{id}", h.DeleteExpense)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/expenses/1", nil)
	req = withUserID(req, 1)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("got %d, want 200", rr.Code)
	}
}

func TestDeleteExpenseHandler_NotFound(t *testing.T) {
	h, _ := setupHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v1/expenses/{id}", h.DeleteExpense)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/expenses/999", nil)
	req = withUserID(req, 1)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("got %d, want 404", rr.Code)
	}
}

func TestDeleteExpenseHandler_WrongUser(t *testing.T) {
	h, repo := setupHandler()
	exp := &domain.Expense{UserID: 1, Category: "Food", Amount: 100}
	repo.Create(context.Background(), exp)

	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /api/v1/expenses/{id}", h.DeleteExpense)

	// user 2 tries to delete user 1's expense
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/expenses/1", nil)
	req = withUserID(req, 2)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("got %d, want 404 (cross-user delete must be denied)", rr.Code)
	}
}

// ─── UpdateExpense ────────────────────────────────────────────────────────────

func TestUpdateExpenseHandler_OK(t *testing.T) {
	h, repo := setupHandler()
	exp := &domain.Expense{UserID: 1, Category: "Food", Amount: 100}
	repo.Create(context.Background(), exp)

	mux := http.NewServeMux()
	mux.HandleFunc("PUT /api/v1/expenses/{id}", h.UpdateExpense)

	body := `{"category":"Transport","amount":250,"comment":"taxi"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/expenses/1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUserID(req, 1)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("got %d, want 200", rr.Code)
	}
	var updated domain.Expense
	decodeJSON(t, rr.Body, &updated)
	if updated.Category != "Transport" || updated.Amount != 250 {
		t.Errorf("unexpected updated values: %+v", updated)
	}
}

func TestUpdateExpenseHandler_NotFound(t *testing.T) {
	h, _ := setupHandler()

	mux := http.NewServeMux()
	mux.HandleFunc("PUT /api/v1/expenses/{id}", h.UpdateExpense)

	body := `{"category":"Food","amount":100}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/expenses/999", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req = withUserID(req, 1)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("got %d, want 404", rr.Code)
	}
}
