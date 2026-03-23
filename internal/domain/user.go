package domain

import "time"

type User struct {
	ID            int       `json:"id"`
	Email         string    `json:"email"`
	PasswordHash  string    `json:"-"`
	Name          string    `json:"name"`
	OAuthProvider string    `json:"oauth_provider,omitempty"`
	OAuthID       string    `json:"oauth_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}
