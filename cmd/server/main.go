package main

import (
	"fmt"
	"log"
	"net/http"

	"go-expense-tracker/internal/handler"
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

	// create service
	svc := service.NewExpenseService(repo)

	// create handler
	apiHandler := handler.NewExpenseHandler(svc)

	// create router
	mux := http.NewServeMux()
	apiHandler.RegisterRoutes(mux)

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Welcome to Expense Tracker API with PostgreSQL!\n")
	})

	// launch server
	port := ":8080"
	fmt.Printf("Server is running on http://localhost%s ...\n", port)

	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatalf("Server crashed: %v", err)
	}
}
