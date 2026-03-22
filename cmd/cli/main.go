package main

import (
	"fmt"
	"log"

	jsonrepo "go-expense-tracker/internal/repository/json"
	"go-expense-tracker/internal/service"
)

func main() {
	repo, err := jsonrepo.NewExpenseRepo("expenses.json")
	if err != nil {
		log.Fatalf("Ошибка при загрузке json файла: %v", err)
	}

	svc := service.NewExpenseService(repo)

	fmt.Println("Данные успешно загружены!")

	RunMenu(svc)
}
