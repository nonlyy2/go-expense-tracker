package main

import (
	"encoding/json"
	"os"
)

const fileName = "expenses.json"

func SaveExpenses(expenses []Expense) error {
	jsonData, err := json.MarshalIndent(expenses, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(fileName, jsonData, 0644); err != nil {
		return err
	}

	return nil
}

func LoadExpenses() ([]Expense, error) {
	data, err := os.ReadFile(fileName)

	if os.IsNotExist(err) {
		return []Expense{}, nil
	} else if err != nil {
		return nil, err
	}

	var loaded []Expense
	err = json.Unmarshal(data, &loaded)
	if err != nil {
		return nil, err
	}

	return loaded, nil
}
