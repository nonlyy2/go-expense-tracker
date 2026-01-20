package main

import (
	"bufio"
	"fmt"
	"os"
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
			exp := inputExpense()
			exp.ID = NextID(expenses)
			expenses = append(expenses, exp)
			SaveExpenses(expenses)
			fmt.Println("✅ Запись добавлена!")

		case "2": // show expenses
			fmt.Println("Твои расходы:\n------------------------------------------------")

			for _, e := range expenses {
				// e — exact expense for each loop step
				fmt.Printf("%d. [%s] %.2f ₸ — %s (%s)\n",
					e.ID, e.Date.Format("2006-01-02"), e.Amount, e.Category, e.Comment)
			}
		case "3": // sum of expenses
			sum_amount := CalculateTotal(expenses)
			fmt.Printf("Сумма расходов: %.2f\n", sum_amount)

		case "4": // update expense
		innerLoop:
			for {
				fmt.Println("    Что хотите изменить?")
				fmt.Println("      1. Категория")
				fmt.Println("      2. Сумма расхода")
				fmt.Println("      3. Доп. информация")
				fmt.Println("      0. Назад в главное меню")
				fmt.Print("Выберите пункт (введите цифру): ")

				choice, err := reader.ReadString('\n')
				if err != nil {
					fmt.Println("Ошибка ввода: ", err)
					continue
				}
				choice = strings.TrimSpace(choice)

				switch choice {
				case "1": // update expense category
					fmt.Print("Введите ID расхода: ")
					id := ScanInt()
					target, err := FindExpenseByID(expenses, id)

					if err != nil {
						fmt.Println("Ошибка:", err)
						continue
					}

					fmt.Print("Введите новую категорию: ")
					target.Category = ScanStr()

				case "2": // update expense amount
					fmt.Print("Введите ID расхода: ")
					id := ScanInt()
					target, err := FindExpenseByID(expenses, id)

					if err != nil {
						fmt.Println("Ошибка:", err)
						continue
					}

					fmt.Print("Введите новую сумму: ")
					target.Amount = ScanFloat()

				case "3": // update expense comment
					fmt.Print("Введите ID расхода: ")
					id := ScanInt()
					target, err := FindExpenseByID(expenses, id)

					if err != nil {
						fmt.Println("Ошибка:", err)
						continue
					}

					fmt.Print("Введите новую доп. инфу: ")
					target.Comment = ScanStr()

				case "0": // выход в основной cli loop
					break innerLoop

				default: // other cases
					fmt.Println("❌ Неверная команда, попробуй еще раз.")
				}
				SaveExpenses(expenses)
				fmt.Println("Успешно обновлено!")
			}

		case "5": // delete expense with id
			fmt.Print("Введите ID расхода для удаления: ")
			id := ScanInt()

			newExpenses, err := DeleteExpenseFromSlice(expenses, id)
			if err != nil {
				fmt.Println("Ошибка:", err)
			} else {
				expenses = newExpenses
				SaveExpenses(expenses)
				fmt.Println("Расход удален!")
			}

		case "0": // exit
			fmt.Println("Пока!")
			return

		default: // other cases
			fmt.Println("❌ Неверная команда, попробуй еще раз.")
		}
	}
}
