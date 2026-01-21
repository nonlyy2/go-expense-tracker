package storage

import (
	"encoding/json"
	"go-expense-tracker/internal/model"
	"os"
)

const fileName = "expenses.json"

func SaveExpenses(expenses []model.Expense) error {
	jsonData, err := json.MarshalIndent(expenses, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(fileName, jsonData, 0644); err != nil {
		return err
	}

	return nil
}

func LoadExpenses() ([]model.Expense, error) {
	data, err := os.ReadFile(fileName)

	if os.IsNotExist(err) {
		return []model.Expense{}, nil
	} else if err != nil {
		return nil, err
	}

	var loaded []model.Expense
	err = json.Unmarshal(data, &loaded)
	if err != nil {
		return nil, err
	}

	return loaded, nil
}
