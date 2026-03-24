package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go-expense-tracker/internal/config"
	"go-expense-tracker/internal/domain"
	"go-expense-tracker/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

type OAuthService interface {
	GetGoogleLoginURL(state string) string
	GetGitHubLoginURL(state string) string
	HandleGoogleCallback(ctx context.Context, code string) (string, error)
	HandleGitHubCallback(ctx context.Context, code string) (string, error)
	IsGoogleConfigured() bool
	IsGitHubConfigured() bool
}

type oauthService struct {
	googleConfig *oauth2.Config
	githubConfig *oauth2.Config
	userRepo     repository.UserRepository
	jwtSecret    string
}

func NewOAuthService(cfg *config.Config, userRepo repository.UserRepository) OAuthService {
	return &oauthService{
		googleConfig: &oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			RedirectURL:  cfg.GoogleRedirectURL,
			Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
			Endpoint:     google.Endpoint,
		},
		githubConfig: &oauth2.Config{
			ClientID:     cfg.GitHubClientID,
			ClientSecret: cfg.GitHubClientSecret,
			RedirectURL:  cfg.GitHubRedirectURL,
			Scopes:       []string{"user:email", "read:user"},
			Endpoint:     github.Endpoint,
		},
		userRepo:  userRepo,
		jwtSecret: cfg.JWTSecret,
	}
}

func (s *oauthService) IsGoogleConfigured() bool {
	return s.googleConfig.ClientID != ""
}

func (s *oauthService) IsGitHubConfigured() bool {
	return s.githubConfig.ClientID != ""
}

func (s *oauthService) GetGoogleLoginURL(state string) string {
	return s.googleConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (s *oauthService) GetGitHubLoginURL(state string) string {
	return s.githubConfig.AuthCodeURL(state)
}

func (s *oauthService) HandleGoogleCallback(ctx context.Context, code string) (string, error) {
	// google access token
	token, err := s.googleConfig.Exchange(ctx, code)
	if err != nil {
		return "", fmt.Errorf("google exchange failed: %w", err)
	}

	// get user data
	client := s.googleConfig.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return "", fmt.Errorf("failed to get google user info: %w", err)
	}
	defer resp.Body.Close()

	var googleUser struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&googleUser); err != nil {
		return "", err
	}

	// search or create user in db, gen JWT token
	return s.processOAuthUser(ctx, "google", googleUser.ID, googleUser.Email, googleUser.Name)
}

func (s *oauthService) HandleGitHubCallback(ctx context.Context, code string) (string, error) {
	token, err := s.githubConfig.Exchange(ctx, code)
	if err != nil {
		return "", fmt.Errorf("github exchange failed: %w", err)
	}

	client := s.githubConfig.Client(ctx, token)

	// get user
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return "", fmt.Errorf("failed to get github user info: %w", err)
	}
	defer resp.Body.Close()

	var githubUser struct {
		ID    int    `json:"id"`
		Login string `json:"login"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&githubUser); err != nil {
		return "", err
	}

	email := githubUser.Email

	// if no access to email
	if email == "" {
		emailsResp, err := client.Get("https://api.github.com/user/emails")
		if err != nil {
			return "", fmt.Errorf("failed to get github emails: %w", err)
		}
		defer emailsResp.Body.Close()

		var emails []struct {
			Email    string `json:"email"`
			Primary  bool   `json:"primary"`
			Verified bool   `json:"verified"`
		}
		if err := json.NewDecoder(emailsResp.Body).Decode(&emails); err != nil {
			return "", err
		}

		for _, e := range emails {
			if e.Primary && e.Verified {
				email = e.Email
				break
			}
		}

		if email == "" {
			return "", fmt.Errorf("no verified primary email found in github account")
		}
	}

	name := githubUser.Name
	if name == "" {
		name = githubUser.Login
	}

	return s.processOAuthUser(ctx, "github", fmt.Sprintf("%d", githubUser.ID), email, name)
}

func (s *oauthService) processOAuthUser(ctx context.Context, provider, oauthID, email, name string) (string, error) {
	// 1. Try to find by OAuth ID (returning user)
	user, err := s.userRepo.GetByOAuth(ctx, provider, oauthID)

	if err == domain.ErrNotFound {
		// 2. Check if user with same email already exists (email-based linking)
		user, err = s.userRepo.GetByEmail(ctx, email)
		if err == nil {
			// Link OAuth to existing account
			if linkErr := s.userRepo.LinkOAuth(ctx, user.ID, provider, oauthID); linkErr != nil {
				return "", fmt.Errorf("failed to link oauth: %w", linkErr)
			}
		} else if err == domain.ErrNotFound {
			// 3. Create brand new OAuth user
			user = &domain.User{
				Email:         email,
				Name:          name,
				OAuthProvider: provider,
				OAuthID:       oauthID,
			}
			if createErr := s.userRepo.CreateOAuth(ctx, user); createErr != nil {
				return "", fmt.Errorf("failed to create oauth user: %w", createErr)
			}
		} else {
			return "", err
		}
	} else if err != nil {
		return "", err
	}

	// gen JWT token for 24 h (claims must match auth_service.VerifyToken)
	claims := jwt.MapClaims{
		"sub": user.ID,
		"exp": time.Now().Add(24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return jwtToken.SignedString([]byte(s.jwtSecret))
}
