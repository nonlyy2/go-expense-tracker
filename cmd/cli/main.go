package main

import (
	"fmt"
	"go-expense-tracker/internal/storage"
)

func main() {
	// upload json files to slices
	expenses, err := storage.LoadExpenses()
	if err != nil {
		fmt.Println("Ошибка при загрузке json файла")
		return
	}
	fmt.Printf("Загружено расходов: %d\n", len(expenses))

	RunMenu(expenses)
}
