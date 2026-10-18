package bot

import (
	"context"
	"expense-tracker/bot/internal/agent"
	"expense-tracker/bot/internal/apiclient"
	"expense-tracker/bot/internal/health"
	"expense-tracker/bot/internal/telegram"
	"log"
	"strings"
	"time"
)

type Bot struct {
	tg      *telegram.Client
	api     *apiclient.Client
	agent   *agent.Agent
	allowed map[int64]bool // пусто = пускать всех
}

func NewBot(tg *telegram.Client, api *apiclient.Client, ag *agent.Agent, allowed map[int64]bool) *Bot {
	return &Bot{tg: tg, api: api, agent: ag, allowed: allowed}
}

// Run опрашивает Telegram (long polling), пока ctx не отменён.
func (b *Bot) Run(ctx context.Context) {
	health.Beat() // сразу при старте, чтобы healthcheck не ждал первого опроса
	offset := 0
	for ctx.Err() == nil {
		updates, err := b.tg.GetUpdates(ctx, offset)
		if err != nil {
			if ctx.Err() == nil {
				log.Printf("getUpdates: %v", err)
				time.Sleep(3 * time.Second)
			}
			continue // при ошибке пульс не обновляется, поэтому через 90 с контейнер станет unhealthy
		}
		health.Beat() // успешный опрос, даже если новых сообщений нет
		for _, u := range updates {
			offset = u.UpdateID + 1
			m := u.Message
			if m == nil || m.From == nil || strings.TrimSpace(m.Text) == "" {
				continue
			}
			go b.handle(ctx, m)
		}
	}
}
