package service

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"go-expense-tracker/internal/domain"
	"go-expense-tracker/internal/repository"
)

type AuthService struct {
	userRepo  repository.UserRepository
	jwtSecret []byte
}

func NewAuthService(userRepo repository.UserRepository, jwtSecret string) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		jwtSecret: []byte(jwtSecret),
	}
}

// Register creates user, hash the pass
func (s *AuthService) Register(ctx context.Context, email, password, name string) (*domain.User, error) {
	if email == "" || len(password) < 6 {
		return nil, domain.ErrInvalidCreds
	}

	// hash pass by DefaultCost
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		Email:        email,
		PasswordHash: string(hashedPassword),
		Name:         name,
	}

	// save to db by repo
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// Login checks pass and issue JWT token
func (s *AuthService) Login(ctx context.Context, email, password string) (string, error) {
	// search by email
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if err == domain.ErrNotFound {
			return "", domain.ErrInvalidCreds // avoid revealing "email not found" to prevent email enumeration
		}
		return "", err
	}

	// compare pass
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", domain.ErrInvalidCreds
	}

	// if pass is valid
	claims := jwt.MapClaims{
		"sub": user.ID,
		"exp": time.Now().Add(24 * time.Hour).Unix(), // token lives 24 hours
		"iat": time.Now().Unix(),                     // token created time
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// VerifyToken checks JWT token then returns ID
func (s *AuthService) VerifyToken(tokenString string) (int, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, domain.ErrUnauthorized
		}
		return s.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return 0, domain.ErrUnauthorized
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, domain.ErrUnauthorized
	}

	sub, ok := claims["sub"].(float64)
	if !ok {
		return 0, domain.ErrUnauthorized
	}

	return int(sub), nil
}

// GetUserByID fetches user info by ID (needed for UI)
func (s *AuthService) GetUserByID(ctx context.Context, id int) (*domain.User, error) {
	return s.userRepo.GetByID(ctx, id)
}
