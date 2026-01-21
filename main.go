package main

import (
	"fmt"
)

func main() {
	// upload json files to slices
	expenses, err := LoadExpenses()
	if err != nil {
		fmt.Println("Ошибка при загрузке json файла")
		return
	}
	fmt.Printf("Загружено расходов: %d\n", len(expenses))

	RunMenu(expenses)
}
