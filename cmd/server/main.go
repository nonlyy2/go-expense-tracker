package main

import (
	"fmt"
	"log"
	"net/http"

	"go-expense-tracker/internal/handler"
	"go-expense-tracker/internal/middleware"
	postgresrepo "go-expense-tracker/internal/repository/postgres"
	"go-expense-tracker/internal/service"
)

func main() {
	fmt.Println("Starting Expense Tracker API with PostgreSQL...")

	// format: postgres://login:pass@host:port/db_name?sslmode=disable
	dsn := "postgres://postgres:qwerty@localhost:5433/expense_tracker?sslmode=disable"

	// create repo to work with PostreSQL
	repo, err := postgresrepo.NewExpenseRepo(dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err := postgresrepo.RunMigrations(repo.DB(), "migrations"); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// create service
	svc := service.NewExpenseService(repo)

	// create handler
	apiHandler := handler.NewExpenseHandler(svc)

	// create router
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Welcome to Expense Tracker API with PostgreSQL!\n")
	})

	// secret JWT key
	jwtSecret := "super-secret-jwt-key-for-my-app"

	db := repo.DB()

	userRepo := postgresrepo.NewUserRepo(db)
	authSvc := service.NewAuthService(userRepo, jwtSecret)
	authHandler := handler.NewAuthHandler(authSvc)

	authHandler.RegisterRoutes(mux)

	authMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return middleware.RequireAuth(authSvc, next)
	}

	// transmit middleware to expense handler
	apiHandler.RegisterRoutes(mux, authMiddleware)

	// launch server
	port := ":8080"
	fmt.Printf("Server is running on http://localhost%s ...\n", port)

	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatalf("Server crashed: %v", err)
	}
}
