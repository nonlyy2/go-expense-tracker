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
		// try to get token
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "missing authorization header", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "invalid authorization header format", http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]

		// check token
		userID, err := authService.VerifyToken(tokenString)
		if err != nil {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}

		// valid token
		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		r = r.WithContext(ctx)

		// trasmit to expense handler
		next.ServeHTTP(w, r)
	}
}

func GetUserID(ctx context.Context) int {
	if id, ok := ctx.Value(UserIDKey).(int); ok {
		return id
	}
	return 0
}
