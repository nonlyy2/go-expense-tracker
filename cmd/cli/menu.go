package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"go-expense-tracker/internal/service"
)

func RunMenu(svc *service.ExpenseService) {
	reader := bufio.NewReader(os.Stdin)
	ctx := context.Background()

	fmt.Println("Добро пожаловать в Expense Tracker (Clean Arch Edition)!")

	for {
		fmt.Println("\nМеню:")
		fmt.Println("  1. Добавить расход")
		fmt.Println("  2. Показать все расходы")
		fmt.Println("  3. Сумма расходов")
		fmt.Println("  4. Обновить расход")
		fmt.Println("  5. Удалить расход")
		fmt.Println("  0. Выход")
		fmt.Print("Выберите пункт (введите цифру): ")

		choice, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Ошибка ввода: ", err)
			continue
		}

		choice = strings.TrimSpace(choice)

		switch choice {
		case "1": // add expense
			fmt.Print("Введите категорию: ")
			cat, _ := reader.ReadString('\n')
			cat = strings.TrimSpace(cat)

			fmt.Print("Введите цену покупки: ")
			amStr, _ := reader.ReadString('\n')
			amount, _ := strconv.ParseFloat(strings.TrimSpace(amStr), 64)

			fmt.Print("Введите доп. информацию: ")
			comm, _ := reader.ReadString('\n')
			comm = strings.TrimSpace(comm)

			_, err := svc.CreateExpense(ctx, cat, amount, comm)
			if err != nil {
				fmt.Printf("❌ Ошибка сохранения: %v\n", err)
			} else {
				fmt.Println("✅ Запись добавлена!")
			}

		case "2": // show expenses
			expenses, err := svc.GetAllExpenses(ctx)
			if err != nil {
				fmt.Println("❌ Ошибка получения данных:", err)
				continue
			}

			fmt.Println("Твои расходы:\n------------------------------------------------")
			for _, e := range expenses {
				fmt.Printf("ID: %d | %s | %.2f тнг | %s | %s\n",
					e.ID, e.Date.Format("02 Jan 15:04"), e.Amount, e.Category, e.Comment)
			}
			fmt.Println("------------------------------------------------")

		case "3": // calculate total
			total, err := svc.GetTotal(ctx)
			if err != nil {
				fmt.Println("❌ Ошибка подсчета:", err)
			} else {
				fmt.Printf("💰 Общая сумма расходов: %.2f\n", total)
			}

		case "4", "5":
			fmt.Println("new CLI in development. use REST API.")

		case "0": // exit
			fmt.Println("Пока!")
			return

		default: // other cases
			fmt.Println("❌ Неверная команда, попробуй еще раз.")
		}
	}
}
