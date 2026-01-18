package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	// upload json files to slices
	expenses, err := LoadExpenses()
	if err != nil {
		fmt.Println("Ошибка при загрузке json файла")
		return
	}
	fmt.Printf("Загружено расходов: %d\n", len(expenses))

	// cli loop
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Добро пожаловать в Expense Tracker!")
	for {
		// список выборов
		fmt.Println("\nМеню:")
		fmt.Println("1. Добавить расход")
		fmt.Println("2. Показать все расходы")
		fmt.Println("3. Сумма расходов")
		fmt.Println("4. Удалить расход")
		fmt.Println("5. Выход")
		fmt.Print("Выберите пункт (введите цифру): ")

		choice, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Ошибка ввода: ", err)
			continue
		}

		choice = strings.TrimSpace(choice)
		switch choice {
		case "1": // add expense
			exp := inputExpense()
			exp.ID = len(expenses) + 1
			expenses = append(expenses, exp)
			SaveExpenses(expenses)
			fmt.Println("✅ Запись добавлена!")

		case "2": // show expenses
			fmt.Println("Твои расходы:\n------------------------------------------------")

			for _, e := range expenses {
				// e — это конкретная трата на текущем шаге цикла
				fmt.Printf("%d. [%s] %.2f ₸ — %s (%s)\n",
					e.ID, e.Date.Format("2006-01-02"), e.Amount, e.Category, e.Comment)
			}
		case "3": // sum of expenses
			sum_amount := 0.00
			fmt.Println("Сумма расходов: ")
			for _, e := range expenses {
				sum_amount += e.Amount
			}
			fmt.Printf("%.2f", sum_amount)

		case "4": // delete expense with id
			fmt.Print("Введите ID расхода для удаления: ")
			idStr, _ := reader.ReadString('\n')
			idStr = strings.TrimSpace(idStr)

			id, err := strconv.Atoi(idStr)
			if err != nil {
				fmt.Println("Ошибка: введите числовой ID")
				continue
			}

			newExpenses, err := DeleteExpenseFromSlice(expenses, id)
			if err != nil {
				fmt.Println("Ошибка:", err)
			} else {
				expenses = newExpenses // update slice
				SaveExpenses(expenses) // save file
				fmt.Println("Расход удален!")
			}

		case "5": // exit
			fmt.Println("Пока!")
			return

		default: // other cases
			fmt.Println("❌ Неверная команда, попробуй еще раз.")
		}
	}
}
