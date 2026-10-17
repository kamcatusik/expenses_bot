package config

import "os"

type Config struct {
	DatabaseURL string
	HTTPAddr    string
	TZ          string
}

func Load() Config {
	return Config{
		DatabaseURL: get("DATABASE_URL", "postgres://app:app@localhost:5432/expenses?sslmode=disable"),
		HTTPAddr:    get("HTTP_ADDR", ":8080"),
		TZ:          get("TZ", "UTC"),
	}
}

func get(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
