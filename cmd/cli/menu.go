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
		fmt.Println("  1. Добавить расход (POST)")
		fmt.Println("  2. Показать все расходы (GET)")
		fmt.Println("  0. Выход")
		fmt.Print("Выберите пункт: ")

		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
			fmt.Print("Введите категорию: ")
			cat, _ := reader.ReadString('\n')
			cat = strings.TrimSpace(cat)

			fmt.Print("Введите цену: ")
			amStr, _ := reader.ReadString('\n')
			amount, _ := strconv.ParseFloat(strings.TrimSpace(amStr), 64)

			fmt.Print("Введите коммент: ")
			comm, _ := reader.ReadString('\n')
			comm = strings.TrimSpace(comm)

			// form query body
			reqBody, _ := json.Marshal(map[string]interface{}{
				"category": cat,
				"amount":   amount,
				"comment":  comm,
			})

			// send post query
			resp, err := http.Post(baseURL, "application/json", bytes.NewBuffer(reqBody))
			if err != nil {
				fmt.Println("❌ Ошибка соединения с сервером. Сервер запущен?")
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusCreated {
				fmt.Println("✅ Запись успешно добавлена в БД!")
			} else {
				body, _ := io.ReadAll(resp.Body)
				fmt.Printf("❌ Ошибка сервера: %s\n", string(body))
			}

		case "2":
			// send get query
			resp, err := http.Get(baseURL)
			if err != nil {
				fmt.Println("❌ Ошибка соединения с сервером. Сервер запущен?")
				continue
			}
			defer resp.Body.Close()

			// read and parse ans
			body, _ := io.ReadAll(resp.Body)
			var expenses []Expense
			if err := json.Unmarshal(body, &expenses); err != nil {
				fmt.Println("❌ Ошибка парсинга данных")
				continue
			}

			fmt.Println("Твои расходы из БД:\n------------------------------------------------")
			for _, e := range expenses {
				fmt.Printf("ID: %d | %s | %.2f тнг | %s | %s\n",
					e.ID, e.Date.Format("02 Jan 15:04"), e.Amount, e.Category, e.Comment)
			}
			fmt.Println("------------------------------------------------")

		case "0":
			fmt.Println("Пока!")
			return

		default:
			fmt.Println("❌ Неверная команда.")
		}
	}
}
