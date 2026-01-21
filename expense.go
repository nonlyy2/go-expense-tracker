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

func RunMenu(expenses []Expense) {
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
