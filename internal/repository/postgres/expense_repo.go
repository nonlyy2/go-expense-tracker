package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"

	"go-expense-tracker/internal/domain"
	"go-expense-tracker/internal/repository"
)

var _ repository.ExpenseRepository = (*expenseRepo)(nil)

type expenseRepo struct {
	db *sql.DB
}

func NewExpenseRepo(dsn string) (*expenseRepo, error) {
	// connect to db
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	// ping
	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &expenseRepo{db: db}, nil
}

// DB returns raw db connection (for migrations etc)
func (r *expenseRepo) DB() *sql.DB {
	return r.db
}

// RunMigrations reads and executes all .up.sql files from migrationsDir in order
func RunMigrations(db *sql.DB, migrationsDir string) error {
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return err
	}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		name := f.Name()
		if len(name) < 7 || name[len(name)-7:] != ".up.sql" {
			continue
		}
		data, err := os.ReadFile(migrationsDir + "/" + name)
		if err != nil {
			return err
		}
		if _, err := db.Exec(string(data)); err != nil {
			return fmt.Errorf("migration %s: %w", name, err)
		}
	}
	return nil
}

func (r *expenseRepo) Create(ctx context.Context, expense *domain.Expense) error {
	// sql query
	query := `INSERT INTO expenses (date, amount, category, comment, user_id) VALUES ($1, $2, $3, $4, $5) RETURNING id`

	// request query and write gen-ed ID
	err := r.db.QueryRowContext(ctx, query, expense.Date, expense.Amount, expense.Category, expense.Comment, expense.UserID).Scan(&expense.ID)
	return err
}

func (r *expenseRepo) GetAll(ctx context.Context, userID int) ([]domain.Expense, error) {
	query := `SELECT id, date, amount, category, comment, user_id FROM expenses WHERE user_id = $1 ORDER BY date DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close() // close read after func, even if error or panic

	var expenses []domain.Expense

	for rows.Next() {
		var e domain.Expense
		if err := rows.Scan(&e.ID, &e.Date, &e.Amount, &e.Category, &e.Comment, &e.UserID); err != nil {
			return nil, err
		}
		expenses = append(expenses, e)
	}

	return expenses, rows.Err()
}

func (r *expenseRepo) GetByID(ctx context.Context, id int, userID int) (*domain.Expense, error) {
	query := `SELECT id, date, amount, category, comment, user_id FROM expenses WHERE id = $1 AND user_id = $2`

	var e domain.Expense
	err := r.db.QueryRowContext(ctx, query, id, userID).Scan(&e.ID, &e.Date, &e.Amount, &e.Category, &e.Comment, &e.UserID)

	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound // map db error to our custom errors
	}
	if err != nil {
		return nil, err
	}

	return &e, nil
}

func (r *expenseRepo) Update(ctx context.Context, expense *domain.Expense) error {
	query := `UPDATE expenses SET amount = $1, category = $2, comment = $3 WHERE id = $4 AND user_id = $5`

	res, err := r.db.ExecContext(ctx, query, expense.Amount, expense.Category, expense.Comment, expense.ID, expense.UserID)
	if err != nil {
		return err
	}

	// if id do not exists
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *expenseRepo) GetMonthlyStats(ctx context.Context, userID int) ([]domain.MonthlyStat, error) {
	query := `SELECT TO_CHAR(date, 'YYYY-MM') AS month, SUM(amount) AS total
	          FROM expenses WHERE user_id = $1
	          GROUP BY month ORDER BY month`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []domain.MonthlyStat
	for rows.Next() {
		var s domain.MonthlyStat
		if err := rows.Scan(&s.Month, &s.Total); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

func (r *expenseRepo) GetMonthlyCategoryStats(ctx context.Context, userID int) ([]domain.MonthCategoryStat, error) {
	query := `SELECT TO_CHAR(date, 'YYYY-MM') AS month, category, SUM(amount) AS total
	          FROM expenses WHERE user_id = $1
	          GROUP BY month, category ORDER BY month, category`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []domain.MonthCategoryStat
	for rows.Next() {
		var s domain.MonthCategoryStat
		if err := rows.Scan(&s.Month, &s.Category, &s.Total); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

func (r *expenseRepo) GetCategoryStats(ctx context.Context, userID int) ([]domain.CategoryStat, error) {
	query := `SELECT category, SUM(amount) AS total, COUNT(*) AS count
	          FROM expenses WHERE user_id = $1
	          GROUP BY category ORDER BY total DESC`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []domain.CategoryStat
	for rows.Next() {
		var s domain.CategoryStat
		if err := rows.Scan(&s.Category, &s.Total, &s.Count); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

func (r *expenseRepo) Delete(ctx context.Context, id int, userID int) error {
	query := `DELETE FROM expenses WHERE id = $1 AND user_id = $2`

	res, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}
