package main

import (
	"fmt"
	"log"
	"net/http"

	"go-expense-tracker/internal/handler"
	jsonrepo "go-expense-tracker/internal/repository/json"
	"go-expense-tracker/internal/service"
)

func main() {
	fmt.Println("Starting Expense Tracker API...")

	// create repo first, and read expenses.json(creates if does not exist)
	repo, err := jsonrepo.NewExpenseRepo("expenses.json")
	if err != nil {
		log.Fatalf("Failed to initialize repository: %v", err)
	}

	// create service
	svc := service.NewExpenseService(repo)

	// create handler
	apiHandler := handler.NewExpenseHandler(svc)

	// create mux(router)
	mux := http.NewServeMux()

	// handler register all routes in router
	apiHandler.RegisterRoutes(mux)

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Welcome to Clean Architecture Expense Tracker API!\nUse /api/v1/expenses")
	})

	// launch server
	port := ":8080"
	fmt.Printf("Server is running on http://localhost%s ...\n", port)

	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatalf("Server crashed: %v", err)
	}
}
