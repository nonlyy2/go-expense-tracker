package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	expenses, err := LoadExpenses()
	if err != nil {
		fmt.Println("Ошибка при загрузке json файла")
		return
	}
	fmt.Printf("Загружено расходов: %d\n", len(expenses))

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Добро пожаловать в Expense Tracker!")

	for {
		fmt.Println("\nМеню:")
		fmt.Println("1. Добавить расход")
		fmt.Println("2. Показать все расходы")
		fmt.Println("3. Выход")
		fmt.Print("Выберите пункт (введите цифру): ")

		choice, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Ошибка ввода: ", err)
			continue
		}

		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
			exp := inputExpense()
			exp.ID = len(expenses) + 1
			expenses = append(expenses, exp)
			SaveExpenses(expenses)
			fmt.Println("✅ Запись добавлена!")

		case "2":
			fmt.Println("Твои расходы:\n------------------------------------------------")

			for _, e := range expenses {
				// e — это конкретная трата на текущем шаге цикла
				fmt.Printf("%d. [%s] %.2f ₸ — %s (%s)\n",
					e.ID, e.Date.Format("2006-01-02"), e.Amount, e.Category, e.Comment)
			}

		case "3":
			fmt.Println("Пока!")
			return
		default:
			fmt.Println("❌ Неверная команда, попробуй еще раз.")
		}
	}
}
