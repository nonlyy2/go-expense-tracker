package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
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

func inputExpense() Expense {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Введите категорию: ")
	cat, _ := reader.ReadString('\n')
	cat = strings.TrimSpace(cat)

	fmt.Print("Введите цену покупки: ")
	amStr, _ := reader.ReadString('\n')
	amStr = strings.TrimSpace(amStr)
	am, err := strconv.ParseFloat(amStr, 64)
	if err != nil {
		fmt.Printf("Ошибка: %v. Установлена сумма 0\n", err)
		am = 0
	}

	fmt.Print("Введите доп. информацию: ")
	comm, _ := reader.ReadString('\n')
	comm = strings.TrimSpace(comm)

	return NewExpense(cat, am, comm)
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

func ScanInt() int {
	reader := bufio.NewReader(os.Stdin)
	for {
		idStr, _ := reader.ReadString('\n')
		idStr = strings.TrimSpace(idStr)
		val, err := strconv.Atoi(idStr)
		if err == nil {
			return val
		}
		fmt.Print("Ошибка, введите число: ")
	}
}

func ScanFloat() float64 {
	reader := bufio.NewReader(os.Stdin)
	for {
		idStr, _ := reader.ReadString('\n')
		idStr = strings.TrimSpace(idStr)
		float, err := strconv.ParseFloat(idStr, 64)
		if err == nil {
			return float
		}
		fmt.Print("Ошибка, введите корректную сумму (например 450.50): ")
	}
}

func ScanStr() string {
	reader := bufio.NewReader(os.Stdin)
	Str, _ := reader.ReadString('\n')
	Str = strings.TrimSpace(Str)

	return Str
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
