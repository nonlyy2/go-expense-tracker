package config

import "os"

type Config struct {
	DatabaseURL string
	ServerPort  string
}

func Load() Config {
	return Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:qwerty@localhost:5433/expense_tracker?sslmode=disable"),
		ServerPort:  getEnv("SERVER_PORT", ":8080"),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}
