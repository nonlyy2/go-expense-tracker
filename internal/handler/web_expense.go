package handler

import (
	"html/template"
	"net/http"
	"strconv"

	"go-expense-tracker/internal/middleware"
	"go-expense-tracker/internal/service"
)

type WebExpenseHandler struct {
	expenseService *service.ExpenseService
	authService    *service.AuthService
	rowTemplate    *template.Template
}

func NewWebExpenseHandler(expenseService *service.ExpenseService, authService *service.AuthService) *WebExpenseHandler {
	rowTmpl := template.Must(template.ParseFiles("templates/partials/expense_row.html"))
	return &WebExpenseHandler{
		expenseService: expenseService,
		authService:    authService,
		rowTemplate:    rowTmpl,
	}
}

// HandleWebLogin accepts form data, sets cookie, returns HX-Redirect
func (h *WebExpenseHandler) HandleWebLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	token, err := h.authService.Login(r.Context(), email, password)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`<p class="text-red-500 text-sm mt-2">Invalid email or password</p>`))
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		HttpOnly: true,
		Path:     "/",
		MaxAge:   86400,
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("HX-Redirect", "/expenses")
	w.WriteHeader(http.StatusOK)
}

// HandleWebRegister accepts form data, registers user, sets cookie
func (h *WebExpenseHandler) HandleWebRegister(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	email := r.FormValue("email")
	password := r.FormValue("password")

	_, err := h.authService.Register(r.Context(), email, password, name)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`<p class="text-red-500 text-sm mt-2">Registration failed: ` + err.Error() + `</p>`))
		return
	}

	token, err := h.authService.Login(r.Context(), email, password)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		HttpOnly: true,
		Path:     "/",
		MaxAge:   86400,
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("HX-Redirect", "/expenses")
	w.WriteHeader(http.StatusOK)
}

// HandleWebLogout clears cookie and redirects to login
func (h *WebExpenseHandler) HandleWebLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		HttpOnly: true,
		Path:     "/",
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// HandleCreate parses form data, creates expense, returns expense_row partial
func (h *WebExpenseHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == 0 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	amount, err := strconv.ParseFloat(r.FormValue("amount"), 64)
	if err != nil {
		http.Error(w, "invalid amount", http.StatusBadRequest)
		return
	}

	category := r.FormValue("category")
	comment := r.FormValue("comment")

	expense, err := h.expenseService.CreateExpense(r.Context(), category, amount, comment, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.rowTemplate.ExecuteTemplate(w, "expense_row", expense)
}

// HandleDelete removes expense and returns empty response for HTMX swap
func (h *WebExpenseHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == 0 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.expenseService.DeleteExpense(r.Context(), id, userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
