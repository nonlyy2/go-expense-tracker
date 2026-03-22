package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"go-expense-tracker/internal/domain"
	"go-expense-tracker/internal/service"
)

type ExpenseHandler struct {
	service *service.ExpenseService
}

func NewExpenseHandler(s *service.ExpenseService) *ExpenseHandler {
	return &ExpenseHandler{service: s}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error()) // 404
	case errors.Is(err, domain.ErrInvalidAmount), errors.Is(err, domain.ErrEmptyCategory):
		writeError(w, http.StatusBadRequest, err.Error()) // 400
	default:
		writeError(w, http.StatusInternalServerError, "internal server error") // 500
	}
}

// ========
// handlers
// ========

// post request
func (h *ExpenseHandler) CreateExpense(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Category string  `json:"category"`
		Amount   float64 `json:"amount"`
		Comment  string  `json:"comment"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	expense, err := h.service.CreateExpense(r.Context(), req.Category, req.Amount, req.Comment)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, expense)
}

// get all expenses
func (h *ExpenseHandler) GetAllExpenses(w http.ResponseWriter, r *http.Request) {
	expenses, err := h.service.GetAllExpenses(r.Context())
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, expenses)
}

// get expense by id
func (h *ExpenseHandler) GetExpenseByID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id format")
		return
	}

	expense, err := h.service.GetExpenseByID(r.Context(), id)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, expense)
}

// put existing expense
func (h *ExpenseHandler) UpdateExpense(w http.ResponseWriter, r *http.Request) {
	// get id from url
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id format")
		return
	}

	// parse request body
	var req struct {
		Category string  `json:"category"`
		Amount   float64 `json:"amount"`
		Comment  string  `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	expense, err := h.service.UpdateExpense(r.Context(), id, req.Category, req.Amount, req.Comment)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, expense)
}

// delete by id
func (h *ExpenseHandler) DeleteExpense(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id format")
		return
	}

	if err := h.service.DeleteExpense(r.Context(), id); err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "expense deleted successfully"})
}

// ==============
// route register
// ==============

// RegisterRoutes binds handler to concrete url-s
func (h *ExpenseHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/expenses", h.CreateExpense)
	mux.HandleFunc("GET /api/v1/expenses", h.GetAllExpenses)
	mux.HandleFunc("GET /api/v1/expenses/{id}", h.GetExpenseByID)
	mux.HandleFunc("PUT /api/v1/expenses/{id}", h.UpdateExpense)
	mux.HandleFunc("DELETE /api/v1/expenses/{id}", h.DeleteExpense)
}
