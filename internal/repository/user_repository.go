package repository

import (
	"context"
	"go-expense-tracker/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id int) (*domain.User, error)

	GetByOAuth(ctx context.Context, provider, oauthID string) (*domain.User, error)
	CreateOAuth(ctx context.Context, user *domain.User) error
	LinkOAuth(ctx context.Context, userID int, provider, oauthID string) error
}
