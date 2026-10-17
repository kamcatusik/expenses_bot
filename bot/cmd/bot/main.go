package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	_ "time/tzdata"

	"expense-tracker/bot/internal/agent"
	"expense-tracker/bot/internal/apiclient"
	"expense-tracker/bot/internal/config"
	"expense-tracker/bot/internal/llm"
	"expense-tracker/bot/internal/telegram"
)

const welcome = "Привет! Я веду учёт расходов. Просто напишите, что потратили: «кофе 350» или «такси 1200 вчера».\n" +
	"Можно спросить «сколько потратил на еду в этом месяце», поправить («последнее было не 350, а 450») или удалить («удали такси»)."

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
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

	heartbeat() // сразу при старте, чтобы healthcheck не ждал первого опроса
	offset := 0
	for ctx.Err() == nil {
		updates, err := tg.GetUpdates(ctx, offset)
		if err != nil {
			if ctx.Err() == nil {
				log.Printf("getUpdates: %v", err)
				time.Sleep(3 * time.Second)
			}
			continue // при ошибке пульс не обновляется, поэтому через 90 с контейнер станет unhealthy
		}
		heartbeat() // успешный опрос, даже если новых сообщений нет
		for _, u := range updates {
			offset = u.UpdateID + 1
			m := u.Message
			if m == nil || m.From == nil || strings.TrimSpace(m.Text) == "" {
				continue
			}
			go handle(ctx, cfg, tg, api, ag, m)
		}
	}
}

const healthFile = "/tmp/healthy"

func heartbeat() { _ = os.WriteFile(healthFile, []byte(time.Now().Format(time.RFC3339)), 0o644) }
func handle(ctx context.Context, cfg config.Config, tg *telegram.Client, api *apiclient.Client, ag *agent.Agent, m *telegram.Message) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	userID := m.From.ID
	if len(cfg.AllowedUsers) > 0 && !cfg.AllowedUsers[userID] {
		_ = tg.SendMessage(ctx, m.Chat.ID, "Нет доступа.")
		return
	}
	text := strings.TrimSpace(m.Text)
	if text == "/start" {
		_ = tg.SendMessage(ctx, m.Chat.ID, welcome)
		return
	}

	if err := api.UpsertUser(ctx, userID, m.From.DisplayName()); err != nil {
		log.Printf("upsert user %d: %v", userID, err)
	}
	tg.SendTyping(ctx, m.Chat.ID)

	reply, err := ag.Handle(ctx, userID, text)
	if err != nil {
		log.Printf("agent (user %d): %v", userID, err)
		reply = "Что-то пошло не так. Попробуйте ещё раз чуть позже."
	}
	if err := tg.SendMessage(ctx, m.Chat.ID, reply); err != nil {
		log.Printf("send (user %d): %v", userID, err)
	}
}
