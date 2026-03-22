package main

import (
	"fmt"
	"log"
	"net/http"

	"go-expense-tracker/internal/config"
	"go-expense-tracker/internal/handler"
	postgresrepo "go-expense-tracker/internal/repository/postgres"
	"go-expense-tracker/internal/service"
)

func main() {
	fmt.Println("Starting Expense Tracker API with PostgreSQL...")

	// load config from env (or defaults)
	cfg := config.Load()

	// connect to db
	repo, err := postgresrepo.NewExpenseRepo(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// run migrations
	if err := postgresrepo.RunMigrations(repo.DB(), "migrations"); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// wire up: service -> handler -> router
	svc := service.NewExpenseService(repo)
	apiHandler := handler.NewExpenseHandler(svc)

	mux := http.NewServeMux()
	apiHandler.RegisterRoutes(mux)

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Welcome to Expense Tracker API with PostgreSQL!\n")
	})

	// launch server
	fmt.Printf("Server is running on http://localhost%s ...\n", cfg.ServerPort)
	if err := http.ListenAndServe(cfg.ServerPort, mux); err != nil {
		log.Fatalf("Server crashed: %v", err)
	}
}
