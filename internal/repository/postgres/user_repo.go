package postgres

import (
	"context"
	"database/sql"

	"go-expense-tracker/internal/domain"
	"go-expense-tracker/internal/repository"
)

var _ repository.UserRepository = (*userRepo)(nil)

type userRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) repository.UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) Create(ctx context.Context, user *domain.User) error {
	query := `INSERT INTO users (email, password_hash, name) 
              VALUES ($1, $2, $3) RETURNING id, created_at`
	err := r.db.QueryRowContext(ctx, query, user.Email, user.PasswordHash, user.Name).Scan(&user.ID, &user.CreatedAt)
	if err != nil {
		return err
	}
	return nil
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT id, email, password_hash, name, oauth_provider, oauth_id, created_at 
              FROM users WHERE email = $1`

	var u domain.User
	var passwordHash, oauthProvider, oauthID sql.NullString

	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&u.ID, &u.Email, &passwordHash, &u.Name, &oauthProvider, &oauthID, &u.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	u.PasswordHash = passwordHash.String
	u.OAuthProvider = oauthProvider.String
	u.OAuthID = oauthID.String
	return &u, nil
}

func (r *userRepo) GetByID(ctx context.Context, id int) (*domain.User, error) {
	query := `SELECT id, email, password_hash, name, oauth_provider, oauth_id, created_at 
              FROM users WHERE id = $1`

	var u domain.User
	var passwordHash, oauthProvider, oauthID sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID, &u.Email, &passwordHash, &u.Name, &oauthProvider, &oauthID, &u.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	u.PasswordHash = passwordHash.String
	u.OAuthProvider = oauthProvider.String
	u.OAuthID = oauthID.String
	return &u, nil
}

func (r *userRepo) GetByOAuth(ctx context.Context, provider, oauthID string) (*domain.User, error) {
	query := `SELECT id, email, password_hash, name, oauth_provider, oauth_id, created_at 
              FROM users WHERE oauth_provider = $1 AND oauth_id = $2`

	var u domain.User
	var passwordHash sql.NullString

	err := r.db.QueryRowContext(ctx, query, provider, oauthID).Scan(
		&u.ID, &u.Email, &passwordHash, &u.Name, &u.OAuthProvider, &u.OAuthID, &u.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	u.PasswordHash = passwordHash.String
	return &u, nil
}

func (r *userRepo) CreateOAuth(ctx context.Context, user *domain.User) error {
	query := `INSERT INTO users (email, name, oauth_provider, oauth_id) 
              VALUES ($1, $2, $3, $4) RETURNING id, created_at`

	err := r.db.QueryRowContext(ctx, query, user.Email, user.Name, user.OAuthProvider, user.OAuthID).
		Scan(&user.ID, &user.CreatedAt)

	if err != nil {
		return err
	}
	return nil
}
