package handler

import (
	"html/template"
	"net/http"
	"sort"
	"strconv"

	"go-expense-tracker/internal/domain"
	"go-expense-tracker/internal/middleware"
	"go-expense-tracker/internal/service"
)

type WebExpenseHandler struct {
	expenseService *service.ExpenseService
	authService    *service.AuthService
	rowTemplate    *template.Template
	listTemplate   *template.Template
}

func NewWebExpenseHandler(expenseService *service.ExpenseService, authService *service.AuthService) *WebExpenseHandler {
	rowTmpl := template.Must(template.ParseFiles("templates/partials/expense_row.html"))
	listTmpl := template.Must(template.ParseFiles(
		"templates/partials/expense_list.html",
		"templates/partials/expense_row.html",
	))
	return &WebExpenseHandler{
		expenseService: expenseService,
		authService:    authService,
		rowTemplate:    rowTmpl,
		listTemplate:   listTmpl,
	}
}

// HandleWebLogin accepts form data, sets cookie, returns HX-Redirect on success
func (h *WebExpenseHandler) HandleWebLogin(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	email := r.FormValue("email")
	password := r.FormValue("password")

	token, err := h.authService.Login(r.Context(), email, password)
	if err != nil {
		// return 200 so htmx swaps the error into target
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
}

// HandleWebRegister validates form data, registers user, auto-logs in
func (h *WebExpenseHandler) HandleWebRegister(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	name := r.FormValue("name")
	email := r.FormValue("email")
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	if password != confirmPassword {
		w.Write([]byte(`<p class="text-red-500 text-sm mt-2">Passwords do not match</p>`))
		return
	}

	_, err := h.authService.Register(r.Context(), email, password, name)
	if err != nil {
		w.Write([]byte(`<p class="text-red-500 text-sm mt-2">` + err.Error() + `</p>`))
		return
	}

	token, err := h.authService.Login(r.Context(), email, password)
	if err != nil {
		w.Header().Set("HX-Redirect", "/login")
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
}

// HandleWebLogout clears cookie and redirects
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

// HandleCreate parses form, creates expense, returns full grouped list
func (h *WebExpenseHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == 0 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	r.ParseForm()
	amount, err := strconv.ParseFloat(r.FormValue("amount"), 64)
	if err != nil {
		http.Error(w, "invalid amount", http.StatusBadRequest)
		return
	}

	if _, err := h.expenseService.CreateExpense(r.Context(), r.FormValue("category"), amount, r.FormValue("comment"), userID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// return full updated list so grouping stays consistent
	expenses, err := h.expenseService.GetAllExpenses(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	sort.Slice(expenses, func(i, j int) bool { return expenses[i].Date.After(expenses[j].Date) })

	h.listTemplate.ExecuteTemplate(w, "expense_list", struct{ DayGroups []DayGroup }{groupByDay(expenses)})
}

// HandleDelete removes expense, returns empty for htmx swap
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

// HandleExpenseList returns filtered/sorted expense_list partial for htmx
func (h *WebExpenseHandler) HandleExpenseList(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == 0 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	expenses, err := h.expenseService.GetAllExpenses(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// filter by category
	if cat := r.URL.Query().Get("category"); cat != "" {
		var filtered []domain.Expense
		for _, e := range expenses {
			if e.Category == cat {
				filtered = append(filtered, e)
			}
		}
		expenses = filtered
	}

	// sort
	switch r.URL.Query().Get("sort") {
	case "date_asc":
		sort.Slice(expenses, func(i, j int) bool { return expenses[i].Date.Before(expenses[j].Date) })
	case "amount_desc":
		sort.Slice(expenses, func(i, j int) bool { return expenses[i].Amount > expenses[j].Amount })
	case "amount_asc":
		sort.Slice(expenses, func(i, j int) bool { return expenses[i].Amount < expenses[j].Amount })
	default: // date_desc
		sort.Slice(expenses, func(i, j int) bool { return expenses[i].Date.After(expenses[j].Date) })
	}

	// for amount sorts pass flat list; for date sorts group by day
	sortParam := r.URL.Query().Get("sort")
	if sortParam == "amount_desc" || sortParam == "amount_asc" {
		h.listTemplate.ExecuteTemplate(w, "expense_list", struct{ Expenses []domain.Expense }{expenses})
		return
	}
	h.listTemplate.ExecuteTemplate(w, "expense_list", struct{ DayGroups []DayGroup }{groupByDay(expenses)})
}
