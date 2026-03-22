package postgres

import (
	"context"
	"database/sql"

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

	// create expense table
	query := `
	CREATE TABLE IF NOT EXISTS expenses (
		id SERIAL PRIMARY KEY,
		date TIMESTAMP NOT NULL,
		amount NUMERIC(10, 2) NOT NULL,
		category VARCHAR(255) NOT NULL,
		comment TEXT
	);`

	_, err = db.Exec(query)
	if err != nil {
		return nil, err
	}

	return &expenseRepo{db: db}, nil
}

func (r *expenseRepo) Create(ctx context.Context, expense *domain.Expense) error {
	// sql query
	query := `INSERT INTO expenses (date, amount, category, comment) VALUES ($1, $2, $3, $4) RETURNING id`

	// request query and write gen-ed ID
	err := r.db.QueryRowContext(ctx, query, expense.Date, expense.Amount, expense.Category, expense.Comment).Scan(&expense.ID)
	return err
}

func (r *expenseRepo) GetAll(ctx context.Context) ([]domain.Expense, error) {
	query := `SELECT id, date, amount, category, comment FROM expenses ORDER BY date DESC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close() // close read after func, even if error or panic

	var expenses []domain.Expense

	for rows.Next() {
		var e domain.Expense
		if err := rows.Scan(&e.ID, &e.Date, &e.Amount, &e.Category, &e.Comment); err != nil {
			return nil, err
		}
		expenses = append(expenses, e)
	}

	return expenses, rows.Err()
}

func (r *expenseRepo) GetByID(ctx context.Context, id int) (*domain.Expense, error) {
	query := `SELECT id, date, amount, category, comment FROM expenses WHERE id = $1`

	var e domain.Expense
	err := r.db.QueryRowContext(ctx, query, id).Scan(&e.ID, &e.Date, &e.Amount, &e.Category, &e.Comment)

	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound // map db error to our custom errors
	}
	if err != nil {
		return nil, err
	}

	return &e, nil
}

func (r *expenseRepo) Update(ctx context.Context, expense *domain.Expense) error {
	query := `UPDATE expenses SET amount = $1, category = $2, comment = $3 WHERE id = $4`

	res, err := r.db.ExecContext(ctx, query, expense.Amount, expense.Category, expense.Comment, expense.ID)
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

func (r *expenseRepo) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM expenses WHERE id = $1`

	res, err := r.db.ExecContext(ctx, query, id)
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
