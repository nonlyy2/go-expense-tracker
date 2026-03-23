package middleware

import (
	"context"
	"net/http"
	"strings"

	"go-expense-tracker/internal/service"
)

// key to store userID
type contextKey string

const UserIDKey contextKey = "userID"

func RequireAuth(authService *service.AuthService, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var tokenString string

		// try cookie first (browser / OAuth flow)
		if cookie, err := r.Cookie("token"); err == nil {
			tokenString = cookie.Value
		}

		// fallback to authorization header (CLI / API clients)
		if tokenString == "" {
			authHeader := r.Header.Get("Authorization")
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1]
			}
		}

		if tokenString == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// check token
		userID, err := authService.VerifyToken(tokenString)
		if err != nil {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func GetUserID(ctx context.Context) int {
	if id, ok := ctx.Value(UserIDKey).(int); ok {
		return id
	}
	return 0
}
