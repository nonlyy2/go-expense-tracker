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
