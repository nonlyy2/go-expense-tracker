package handler

import (
	"net/http"

	"go-expense-tracker/internal/middleware"
	"go-expense-tracker/internal/service"
)

type StatsHandler struct {
	service *service.ExpenseService
}

func NewStatsHandler(s *service.ExpenseService) *StatsHandler {
	return &StatsHandler{service: s}
}

func (h *StatsHandler) HandleMonthlyStats(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	stats, err := h.service.GetMonthlyStats(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (h *StatsHandler) HandleCategoryStats(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	stats, err := h.service.GetCategoryStats(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (h *StatsHandler) HandleMonthlyCategoryStats(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == 0 {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	stats, err := h.service.GetMonthlyCategoryStats(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (h *StatsHandler) RegisterRoutes(mux *http.ServeMux, requireAuth func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("GET /api/v1/stats/monthly", requireAuth(h.HandleMonthlyStats))
	mux.HandleFunc("GET /api/v1/stats/by-category", requireAuth(h.HandleCategoryStats))
	mux.HandleFunc("GET /api/v1/stats/monthly-by-category", requireAuth(h.HandleMonthlyCategoryStats))
}
