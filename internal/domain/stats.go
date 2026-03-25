package domain

type MonthlyStat struct {
	Month string  `json:"month"` // "2025-01"
	Total float64 `json:"total"`
}

type CategoryStat struct {
	Category string  `json:"category"`
	Total    float64 `json:"total"`
	Count    int     `json:"count"`
}

// MonthCategoryStat is used for stacked bar chart (month + category breakdown).
type MonthCategoryStat struct {
	Month    string  `json:"month"`    // "2025-01"
	Category string  `json:"category"`
	Total    float64 `json:"total"`
}
