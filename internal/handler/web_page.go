package handler

import (
	"html/template"
	"net/http"
	"sort"

	"go-expense-tracker/internal/domain"
	"go-expense-tracker/internal/middleware"
	"go-expense-tracker/internal/service"
)

type WebPageHandler struct {
	templates      map[string]*template.Template
	expenseService *service.ExpenseService
	authService    *service.AuthService
	oauthService   service.OAuthService
}

func NewWebPageHandler(
	expenseService *service.ExpenseService,
	authService *service.AuthService,
	oauthService service.OAuthService,
) *WebPageHandler {
	templates := make(map[string]*template.Template)

	pages := []string{"login.html", "register.html", "expenses.html", "dashboard.html"}

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
		oauthService:   oauthService,
	}
}

type PageData struct {
	User        map[string]interface{}
	Expenses    []domain.Expense
	DayGroups   []DayGroup
	Categories  []string
	Summary     []CategorySummary
	GoogleOAuth bool
	GitHubOAuth bool
}

type CategorySummary struct {
	Category string
	Total    float64
	Count    int
}

type DayGroup struct {
	Day        string // e.g. "Monday, 15 April"
	MonthLabel string // e.g. "April 2025" — non-empty only when month changes
	Total      float64
	Expenses   []domain.Expense
}

// groupByDay groups an already-sorted expense slice into day buckets.
// MonthLabel is set only on the first group of each month.
func groupByDay(expenses []domain.Expense) []DayGroup {
	var groups []DayGroup
	var prevKey, prevMonth string

	for _, e := range expenses {
		key := e.Date.Format("2006-01-02")
		if key != prevKey {
			month := e.Date.Format("January 2006")
			monthLabel := ""
			if month != prevMonth {
				monthLabel = month
				prevMonth = month
			}
			groups = append(groups, DayGroup{
				Day:        e.Date.Format("Monday, 2 January"),
				MonthLabel: monthLabel,
			})
			prevKey = key
		}
		g := &groups[len(groups)-1]
		g.Total += e.Amount
		g.Expenses = append(g.Expenses, e)
	}
	return groups
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
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (h *WebPageHandler) HandleDashboard(w http.ResponseWriter, r *http.Request) {
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
	h.render(w, "dashboard.html", PageData{
		User: map[string]interface{}{"Name": user.Name},
	})
}

func (h *WebPageHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	h.render(w, "login.html", PageData{
		GoogleOAuth: h.oauthService.IsGoogleConfigured(),
		GitHubOAuth: h.oauthService.IsGitHubConfigured(),
	})
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

	// build category list and summary
	catTotals := make(map[string]*CategorySummary)
	for _, e := range expenses {
		cs, ok := catTotals[e.Category]
		if !ok {
			cs = &CategorySummary{Category: e.Category}
			catTotals[e.Category] = cs
		}
		cs.Total += e.Amount
		cs.Count++
	}

	var summary []CategorySummary
	var categories []string
	for _, cs := range catTotals {
		summary = append(summary, *cs)
		categories = append(categories, cs.Category)
	}
	sort.Slice(summary, func(i, j int) bool { return summary[i].Total > summary[j].Total })
	sort.Strings(categories)

	// default sort: newest first
	sort.Slice(expenses, func(i, j int) bool { return expenses[i].Date.After(expenses[j].Date) })

	h.render(w, "expenses.html", PageData{
		User:       map[string]interface{}{"Name": user.Name},
		DayGroups:  groupByDay(expenses),
		Categories: categories,
		Summary:    summary,
	})
}
