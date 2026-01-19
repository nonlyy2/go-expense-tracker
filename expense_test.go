package main

import "testing"

func TestCalculateTotal(t *testing.T) {
	tests := []struct {
		name     string
		input    []Expense
		expected float64
	}{
		{
			name: "test expense",
			input: []Expense{
				{Amount: 450.00},
			},
			expected: 450.00,
		},
		{
			name: "test expense",
			input: []Expense{
				{Amount: 450.00},
				{Amount: 1530.00},
				{Amount: 2660.00},
			},
			expected: 4640.00,
		},
		{
			name:     "test expense",
			input:    []Expense{},
			expected: 0.00,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateTotal(tt.input)
			if got != tt.expected {
				t.Errorf("Ожидалось: %.2f\nПолучили: %.2f", tt.expected, got)
			}
		})
	}
}
