package bot

import (
	"context"
	"expense-tracker/bot/internal/agent"
	"expense-tracker/bot/internal/apiclient"
	"expense-tracker/bot/internal/health"
	"expense-tracker/bot/internal/telegram"
	"log"
	"strings"
	"sync"
	"time"
)

type Bot struct {
	tg      *telegram.Client
	api     *apiclient.Client
	agent   *agent.Agent
	allowed map[int64]bool // пусто = пускать всех
	wg      sync.WaitGroup
}

func NewBot(tg *telegram.Client, api *apiclient.Client, ag *agent.Agent, allowed map[int64]bool) *Bot {
	return &Bot{tg: tg, api: api, agent: ag, allowed: allowed}
}

// Run опрашивает Telegram (long polling), пока ctx не отменён.
func (b *Bot) Run(ctx context.Context) {
	health.Beat() // сразу при старте, чтобы healthcheck не ждал первого опроса
	offset := 0
	sem := make(chan struct{}, 10) // ограничиваем одновременную обработку до 10 сообщений
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

			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				break
			}

			b.wg.Add(1)
			go func(msg *telegram.Message) {
				defer func() {
					<-sem
					b.wg.Done()
				}()
				// Используем context.Background(), чтобы при отмене контекста polling'а
				// уже запущенная обработка сообщения успела корректно завершиться
				b.handle(context.Background(), msg)
			}(m)
		}
	}

	log.Println("bot stopping: waiting for in-flight requests...")
	done := make(chan struct{})
	go func() {
		b.wg.Wait()
		close(done)
	}()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	select {
	case <-done:
		log.Println("bot stopped gracefully")
	case <-shutdownCtx.Done():
		log.Println("bot shutdown timed out: forced exit")
	}
}
