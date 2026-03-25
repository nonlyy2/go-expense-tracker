package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/joho/godotenv"

	"go-expense-tracker/internal/domain"
	postgresrepo "go-expense-tracker/internal/repository/postgres"
	"go-expense-tracker/internal/service"
)

type catConfig struct {
	name     string
	weight   int     // relative frequency out of 100
	minAmt   float64 // typical min per transaction
	maxAmt   float64 // typical max per transaction
	comments []string
}

var cats = []catConfig{
	{
		name: "Food", weight: 35, minAmt: 300, maxAmt: 4000,
		comments: []string{"lunch", "groceries", "coffee", "dinner", "snacks", "breakfast", "food delivery"},
	},
	{
		name: "Transport", weight: 20, minAmt: 150, maxAmt: 3000,
		comments: []string{"taxi", "bus pass", "fuel", "metro", "parking", "car service"},
	},
	{
		name: "Utilities", weight: 8, minAmt: 3000, maxAmt: 45000,
		comments: []string{"electricity", "water", "internet", "phone plan", "rent"},
	},
	{
		name: "Shopping", weight: 12, minAmt: 1000, maxAmt: 25000,
		comments: []string{"clothes", "electronics", "gifts", "home goods", "accessories"},
	},
	{
		name: "Health", weight: 6, minAmt: 800, maxAmt: 15000,
		comments: []string{"pharmacy", "gym membership", "doctor visit", "vitamins", "insurance"},
	},
	{
		name: "Entertainment", weight: 9, minAmt: 500, maxAmt: 8000,
		comments: []string{"cinema", "concert", "games", "streaming", "books", "hobby"},
	},
	{
		name: "Education", weight: 5, minAmt: 2000, maxAmt: 20000,
		comments: []string{"online course", "textbooks", "tutoring", "software license", "exam fee"},
	},
	{
		name: "Other", weight: 5, minAmt: 200, maxAmt: 5000,
		comments: []string{"misc", "donation", "bank fee", "repair", "subscription"},
	},
}

// weightedPick picks a category index using weighted random selection.
func weightedPick(rng *rand.Rand) int {
	total := 0
	for _, c := range cats {
		total += c.weight
	}
	r := rng.Intn(total)
	for i, c := range cats {
		r -= c.weight
		if r < 0 {
			return i
		}
	}
	return len(cats) - 1
}

func main() {
	godotenv.Load()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:qwerty@localhost:5433/expense_tracker?sslmode=disable"
	}

	repo, err := postgresrepo.NewExpenseRepo(dsn)
	if err != nil {
		log.Fatalf("db connection failed: %v", err)
	}
	db := repo.DB()
	if err := postgresrepo.RunMigrations(db, "migrations"); err != nil {
		log.Fatalf("migrations failed: %v", err)
	}

	userRepo := postgresrepo.NewUserRepo(db)
	authSvc := service.NewAuthService(userRepo, "super-secret-dev-key")
	ctx := context.Background()

	email, password := "demo@demo.com", "Demo1234!"
	if _, err = authSvc.Register(ctx, email, password, "Demo User"); err != nil {
		fmt.Printf("user may already exist: %v\n", err)
	}

	token, err := authSvc.Login(ctx, email, password)
	if err != nil {
		log.Fatalf("login failed: %v", err)
	}
	userID, err := authSvc.VerifyToken(token)
	if err != nil {
		log.Fatalf("verify token failed: %v", err)
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	now := time.Now()
	count := 0

	for m := 11; m >= 0; m-- {
		monthStart := time.Date(now.Year(), now.Month()-time.Month(m), 1, 0, 0, 0, 0, time.Local)
		daysInMonth := time.Date(monthStart.Year(), monthStart.Month()+1, 0, 0, 0, 0, 0, time.Local).Day()

		// realistic: 60-100 transactions/month, heavier in recent months
		base := 60
		if m <= 2 {
			base = 90
		}
		numExpenses := base + rng.Intn(40)

		for i := 0; i < numExpenses; i++ {
			idx := weightedPick(rng)
			cat := cats[idx]
			comment := cat.comments[rng.Intn(len(cat.comments))]

			// amount: log-normal-ish feel — skew toward lower values
			spread := cat.maxAmt - cat.minAmt
			amount := cat.minAmt + spread*(rng.Float64()*rng.Float64()) // skewed right
			amount = float64(int(amount*100)) / 100

			day := 1 + rng.Intn(daysInMonth)
			hour := 8 + rng.Intn(14)
			date := time.Date(
				monthStart.Year(), monthStart.Month(), day,
				hour, rng.Intn(60), 0, 0, time.Local,
			)

			exp := &domain.Expense{
				UserID:   userID,
				Date:     date,
				Amount:   amount,
				Category: cat.name,
				Comment:  comment,
			}
			if err := repo.Create(ctx, exp); err != nil {
				log.Printf("failed to create expense: %v", err)
				continue
			}
			count++
		}
	}

	fmt.Printf("seeded %d expenses\n", count)
	fmt.Printf("login: %s / %s\n", email, password)
}
