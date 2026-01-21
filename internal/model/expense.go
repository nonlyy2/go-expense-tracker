package model

import (
	"fmt"
	"time"
)

// Expense - это структура, описывающая одну трату
type Expense struct {
	ID       int       `json:"id"`
	Date     time.Time `json:"date"`
	Amount   float64   `json:"amount"`
	Category string    `json:"category"`
	Comment  string    `json:"comment"`
}

func NewExpense(category string, amount float64, comment string) Expense {
	newExpense := Expense{
		ID:       1,
		Date:     time.Now(),
		Amount:   amount,
		Category: category,
		Comment:  comment,
	}
	return newExpense
}

func CalculateTotal(expenses []Expense) float64 {
	var total float64
	for _, e := range expenses {
		total += e.Amount
	}
	return total
}

func NextID(expenses []Expense) int {
	maxid := 0
	for _, e := range expenses {
		if e.ID > maxid {
			maxid = e.ID
		}
	}
	return maxid + 1
}

func FindExpenseByID(expenses []Expense, id int) (*Expense, error) {
	for i := range expenses {
		if expenses[i].ID == id {
			return &expenses[i], nil // return pointer to exact expense
		}
	}
	return nil, fmt.Errorf("расход с ID %d не найден", id)
}

func DeleteExpenseFromSlice(expenses []Expense, id int) ([]Expense, error) {
	index := -1
	for i, e := range expenses {
		if e.ID == id {
			index = i
			break
		}
	}

	if index == -1 {
		return expenses, fmt.Errorf("Расход с ID %d не найден", id)
	}

	return append(expenses[:index], expenses[index+1:]...), nil
}
