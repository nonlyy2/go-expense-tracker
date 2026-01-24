package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

type Expense struct {
	ID       int     `json:"id"`
	Category string  `json:"category"`
	Amount   float64 `json:"amount"`
}

var expenses = []Expense{
	{ID: 1, Category: "Такси до NU", Amount: 2000.0},
	{ID: 2, Category: "Кофе перед митингом", Amount: 1800.0},
	{ID: 3, Category: "Подписка на AI", Amount: 5000.0},
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /expenses", createExpensesHandler)
	mux.HandleFunc("GET /expenses", getExpensesHandler)
	mux.HandleFunc("GET /expenses/{id}", getExpenseByIDHandler)
	mux.HandleFunc("PUT /expenses/{id}", updateExpenseHandler)
	mux.HandleFunc("DELETE /expenses/{id}", deleteExpenseHandler)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Halo, my server is working now!\nGo to /expenses to see the expenses")
	})

	fmt.Println("Server is running on http://localhost:8080 ...")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		panic(err)
	}
}

// create expense (post)
func createExpensesHandler(w http.ResponseWriter, r *http.Request) {
	var newExpense Expense
	if err := json.NewDecoder(r.Body).Decode(&newExpense); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	newExpense.ID = len(expenses) + 1
	expenses = append(expenses, newExpense)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newExpense)
}

// show all expenses (get)
func getExpensesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(expenses)
}

// show expense by exact id (get)
func getExpenseByIDHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	for _, expense := range expenses {
		if expense.ID == id {
			w.Header().Set("Content-type", "application/json")
			json.NewEncoder(w).Encode(expense)
			return
		}
	}

	http.Error(w, "Expense not found", http.StatusNotFound)
}

// update expense by id (put)
func updateExpenseHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var updatedExpense Expense
	if err := json.NewDecoder(r.Body).Decode(&updatedExpense); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	for i, expense := range expenses {
		if expense.ID == id {
			updatedExpense.ID = id
			expenses[i] = updatedExpense

			w.Header().Set("Content-type", "application/json")
			json.NewEncoder(w).Encode(updatedExpense)
			return
		}
	}

	http.Error(w, "Expense not found", http.StatusNotFound)
}

// delete expense by id (delete)
func deleteExpenseHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	for i, expense := range expenses {
		if expense.ID == id {
			expenses = append(expenses[:i], expenses[i+1:]...)
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}

	http.Error(w, "Expense not found", http.StatusNotFound)
}
