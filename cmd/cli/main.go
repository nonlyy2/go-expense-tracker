package main

import (
	"fmt"
)

const baseURL = "http://localhost:8080/api/v1/expenses"

func main() {
	fmt.Println("Запуск CLI HTTP-клиента...")
	RunMenu()
}
