package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	TelegramToken string
	LLMBaseURL    string
	LLMAPIKey     string
	LLMModel      string
	BackendURL    string
	TZ            string
	// AllowedUsers — необязательный белый список Telegram ID. Пусто = пускать всех.
	AllowedUsers map[int64]bool
}

func Load() (Config, error) {
	c := Config{
		TelegramToken: os.Getenv("TELEGRAM_TOKEN"),
		LLMBaseURL:    strings.TrimRight(get("LLM_BASE_URL", "https://api.openai.com/v1"), "/"),
		LLMAPIKey:     os.Getenv("LLM_API_KEY"),
		LLMModel:      get("LLM_MODEL", "gpt-4o-mini"),
		BackendURL:    strings.TrimRight(get("BACKEND_URL", "http://backend:8080"), "/"),
		TZ:            get("TZ", "UTC"),
		AllowedUsers:  map[int64]bool{},
	}
	if c.TelegramToken == "" {
		return c, fmt.Errorf("TELEGRAM_TOKEN is required")
	}
	for _, s := range strings.Split(os.Getenv("ALLOWED_USER_IDS"), ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return c, fmt.Errorf("ALLOWED_USER_IDS: bad id %q", s)
		}
		c.AllowedUsers[id] = true
	}
	return c, nil
}

func get(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
