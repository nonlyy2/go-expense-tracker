package config

import (
	"os"
)

type Config struct {
	DBHost             string
	DBPort             string
	DBUser             string
	DBPassword         string
	DBName             string
	ServerPort         string
	JWTSecret          string
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	GitHubClientID     string
	GitHubClientSecret string
	GitHubRedirectURL  string
}

func LoadConfig() *Config {
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "super-secret-dev-key" // for local dev
	}

	return &Config{
		DBHost:     os.Getenv("DB_HOST"),
		DBPort:     os.Getenv("DB_PORT"),
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName:     os.Getenv("DB_NAME"),
		ServerPort: port,
		JWTSecret:  jwtSecret,

		// OAuth settings
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:  getEnvOrDefault("GOOGLE_REDIRECT_URL", "http://localhost:8080/auth/google/callback"),

		GitHubClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		GitHubClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		GitHubRedirectURL:  getEnvOrDefault("GITHUB_REDIRECT_URL", "http://localhost:8080/auth/github/callback"),
	}
}

func getEnvOrDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
