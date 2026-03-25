package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"go-expense-tracker/internal/config"
	"go-expense-tracker/internal/handler"
	"go-expense-tracker/internal/middleware"
	postgresrepo "go-expense-tracker/internal/repository/postgres"
	"go-expense-tracker/internal/service"
)

func main() {
	// load .env if present (no error if missing — prod uses real env vars)
	godotenv.Load()

	fmt.Println("Starting Expense Tracker...")

	cfg := config.LoadConfig()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:qwerty@localhost:5433/expense_tracker?sslmode=disable"
	}

	// DB connection and migrations
	repo, err := postgresrepo.NewExpenseRepo(dsn)
	if err != nil {
		log.Fatalf("DB connection failed: %v", err)
	}
	db := repo.DB()
	if err := postgresrepo.RunMigrations(db, "migrations"); err != nil {
		log.Fatalf("Migrations failed: %v", err)
	}

	// Repos and services
	userRepo := postgresrepo.NewUserRepo(db)
	svc := service.NewExpenseService(repo)
	authSvc := service.NewAuthService(userRepo, cfg.JWTSecret)
	oauthService := service.NewOAuthService(cfg, userRepo)

	// Handlers
	apiHandler := handler.NewExpenseHandler(svc)
	authHandler := handler.NewAuthHandler(authSvc)
	oauthHandler := handler.NewOAuthHandler(oauthService)
	statsHandler := handler.NewStatsHandler(svc)
	webPageHandler := handler.NewWebPageHandler(svc, authSvc, oauthService)
	webExpenseHandler := handler.NewWebExpenseHandler(svc, authSvc)

	// Auth middleware
	authMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return middleware.RequireAuth(authSvc, next)
	}

	mux := http.NewServeMux()

	// Web pages
	mux.HandleFunc("GET /{$}", webPageHandler.HandleIndex)
	mux.HandleFunc("GET /login", webPageHandler.HandleLogin)
	mux.HandleFunc("GET /register", webPageHandler.HandleRegister)
	mux.HandleFunc("GET /dashboard", authMiddleware(webPageHandler.HandleDashboard))
	mux.HandleFunc("GET /expenses", authMiddleware(webPageHandler.HandleExpenses))

	// Web form handlers (HTMX)
	mux.HandleFunc("POST /web/auth/login", webExpenseHandler.HandleWebLogin)
	mux.HandleFunc("POST /web/auth/register", webExpenseHandler.HandleWebRegister)
	mux.HandleFunc("POST /web/auth/logout", webExpenseHandler.HandleWebLogout)
	mux.HandleFunc("POST /expenses", authMiddleware(webExpenseHandler.HandleCreate))
	mux.HandleFunc("DELETE /expenses/{id}", authMiddleware(webExpenseHandler.HandleDelete))
	mux.HandleFunc("GET /expenses/list", authMiddleware(webExpenseHandler.HandleExpenseList))

	// OAuth
	mux.HandleFunc("GET /auth/google/login", oauthHandler.HandleGoogleLogin)
	mux.HandleFunc("GET /auth/google/callback", oauthHandler.HandleGoogleCallback)
	mux.HandleFunc("GET /auth/github/login", oauthHandler.HandleGitHubLogin)
	mux.HandleFunc("GET /auth/github/callback", oauthHandler.HandleGitHubCallback)

	// JSON API
	authHandler.RegisterRoutes(mux)
	apiHandler.RegisterRoutes(mux, authMiddleware)
	statsHandler.RegisterRoutes(mux, authMiddleware)

	port := ":" + cfg.ServerPort
	if cfg.ServerPort == "" {
		port = ":8080"
	}

	// rate limiter: 10 req/sec, burst 20
	rl := middleware.NewRateLimiter(10, 20)

	srv := &http.Server{
		Addr:    port,
		Handler: middleware.Logging(rl.Middleware(mux)),
	}

	go func() {
		fmt.Printf("Server running on http://localhost%s\n", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server crashed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}
	fmt.Println("server stopped gracefully")
}
