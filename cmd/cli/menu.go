package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Expense struct {
	ID       int       `json:"id"`
	Date     time.Time `json:"date"`
	Amount   float64   `json:"amount"`
	Category string    `json:"category"`
	Comment  string    `json:"comment"`
}

func RunMenu() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Добро пожаловать в Expense Tracker!")

	for {
		fmt.Println("\nМеню:")
		fmt.Println("  1. Добавить расход")
		fmt.Println("  2. Показать все расходы")
		fmt.Println("  3. Сумма расходов")
		fmt.Println("  4. Обновить расход")
		fmt.Println("  5. Удалить расход")
		fmt.Println("  0. Выход")
		fmt.Print("Выберите пункт: ")

		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
			addExpense(reader)
		case "2":
			showAllExpenses()
		case "3":
			showTotal()
		case "4":
			updateExpense(reader)
		case "5":
			deleteExpense(reader)
		case "0":
			fmt.Println("Пока!")
			return
		default:
			fmt.Println("❌ Неверная команда.")
		}
	}
}

func addExpense(reader *bufio.Reader) {
	fmt.Print("Введите категорию: ")
	cat, _ := reader.ReadString('\n')
	cat = strings.TrimSpace(cat)

	fmt.Print("Введите цену: ")
	amStr, _ := reader.ReadString('\n')
	amount, _ := strconv.ParseFloat(strings.TrimSpace(amStr), 64)

	fmt.Print("Введите коммент: ")
	comm, _ := reader.ReadString('\n')
	comm = strings.TrimSpace(comm)

	reqBody, _ := json.Marshal(map[string]interface{}{
		"category": cat,
		"amount":   amount,
		"comment":  comm,
	})

	resp, err := http.Post(baseURL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		fmt.Println("❌ Ошибка соединения с сервером. Сервер запущен?")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		fmt.Println("✅ Запись успешно добавлена!")
	} else {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("❌ Ошибка сервера: %s\n", string(body))
	}
}

func showAllExpenses() {
	resp, err := http.Get(baseURL)
	if err != nil {
		fmt.Println("❌ Ошибка соединения с сервером. Сервер запущен?")
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var expenses []Expense
	if err := json.Unmarshal(body, &expenses); err != nil {
		fmt.Println("❌ Ошибка парсинга данных")
		return
	}

	if len(expenses) == 0 {
		fmt.Println("Расходов пока нет.")
		return
	}

	fmt.Println("Твои расходы:\n------------------------------------------------")
	for _, e := range expenses {
		fmt.Printf("ID: %d | %s | %.2f тнг | %s | %s\n",
			e.ID, e.Date.Format("02 Jan 15:04"), e.Amount, e.Category, e.Comment)
	}
	fmt.Println("------------------------------------------------")
}

func showTotal() {
	resp, err := http.Get(baseURL)
	if err != nil {
		fmt.Println("❌ Ошибка соединения с сервером. Сервер запущен?")
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var expenses []Expense
	if err := json.Unmarshal(body, &expenses); err != nil {
		fmt.Println("❌ Ошибка парсинга данных")
		return
	}

	var total float64
	for _, e := range expenses {
		total += e.Amount
	}
	fmt.Printf("Общая сумма расходов: %.2f тнг\n", total)
}

func updateExpense(reader *bufio.Reader) {
	fmt.Print("Введите ID расхода: ")
	idStr, _ := reader.ReadString('\n')
	id := strings.TrimSpace(idStr)

	fmt.Print("Новая категория: ")
	cat, _ := reader.ReadString('\n')
	cat = strings.TrimSpace(cat)

	fmt.Print("Новая цена: ")
	amStr, _ := reader.ReadString('\n')
	amount, _ := strconv.ParseFloat(strings.TrimSpace(amStr), 64)

	fmt.Print("Новый коммент: ")
	comm, _ := reader.ReadString('\n')
	comm = strings.TrimSpace(comm)

	reqBody, _ := json.Marshal(map[string]interface{}{
		"category": cat,
		"amount":   amount,
		"comment":  comm,
	})

	req, _ := http.NewRequest(http.MethodPut, baseURL+"/"+id, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("❌ Ошибка соединения с сервером. Сервер запущен?")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Println("✅ Расход обновлён!")
	} else {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("❌ Ошибка: %s\n", string(body))
	}
}

func deleteExpense(reader *bufio.Reader) {
	fmt.Print("Введите ID расхода для удаления: ")
	idStr, _ := reader.ReadString('\n')
	id := strings.TrimSpace(idStr)

	req, _ := http.NewRequest(http.MethodDelete, baseURL+"/"+id, nil)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("❌ Ошибка соединения с сервером. Сервер запущен?")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Println("✅ Расход удалён!")
	} else {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("❌ Ошибка: %s\n", string(body))
	}
}
