package handler

import (
	"html/template"
	"net/http"

	"go-expense-tracker/internal/middleware"
	"go-expense-tracker/internal/service"
)

type WebPageHandler struct {
	templates      map[string]*template.Template
	expenseService *service.ExpenseService
	authService    *service.AuthService
}

func NewWebPageHandler(expenseService *service.ExpenseService, authService *service.AuthService) *WebPageHandler {
	templates := make(map[string]*template.Template)

	pages := []string{"login.html", "register.html", "expenses.html"}

	for _, page := range pages {
		files := []string{
			"templates/layouts/base.html",
			"templates/pages/" + page,
		}

		t := template.Must(template.New("base.html").ParseFiles(files...))
		t = template.Must(t.ParseGlob("templates/partials/*.html"))

		templates[page] = t
	}

	return &WebPageHandler{
		templates:      templates,
		expenseService: expenseService,
		authService:    authService,
	}
}

type PageData struct {
	User     map[string]interface{}
	Expenses interface{}
}

func (h *WebPageHandler) render(w http.ResponseWriter, page string, data interface{}) {
	tmpl, ok := h.templates[page]
	if !ok {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}
	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *WebPageHandler) HandleIndex(w http.ResponseWriter, r *http.Request) {
	if _, err := r.Cookie("token"); err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/expenses", http.StatusSeeOther)
}

func (h *WebPageHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	h.render(w, "login.html", nil)
}

func (h *WebPageHandler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	h.render(w, "register.html", nil)
}

func (h *WebPageHandler) HandleExpenses(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	user, err := h.authService.GetUserByID(r.Context(), userID)
	if err != nil {
		http.SetCookie(w, &http.Cookie{Name: "token", MaxAge: -1, Path: "/"})
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	expenses, err := h.expenseService.GetAllExpenses(r.Context(), userID)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	h.render(w, "expenses.html", PageData{
		User: map[string]interface{}{
			"Name": user.Name,
		},
		Expenses: expenses,
	})
}
