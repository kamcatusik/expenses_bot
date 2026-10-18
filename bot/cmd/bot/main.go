package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"expense-tracker/bot/internal/agent"
	"expense-tracker/bot/internal/apiclient"
	"expense-tracker/bot/internal/bot"
	"expense-tracker/bot/internal/config"
	"expense-tracker/bot/internal/llm"
	"expense-tracker/bot/internal/logx"
	"expense-tracker/bot/internal/telegram"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	logx.Setup(cfg.TelegramToken, cfg.LLMAPIKey)

	loc, err := time.LoadLocation(cfg.TZ)
	if err != nil {
		log.Fatalf("bad TZ %q: %v", cfg.TZ, err)
	}

	tg := telegram.NewClient(cfg.TelegramToken)
	api := apiclient.NewClient(cfg.BackendURL)
	ag := agent.New(llm.NewClient(cfg.LLMBaseURL, cfg.LLMAPIKey, cfg.LLMModel), api, loc)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Printf("bot started (model=%s, llm=%s)", cfg.LLMModel, cfg.LLMBaseURL)
	bot.NewBot(tg, api, ag, cfg.AllowedUsers).Run(ctx)
}
