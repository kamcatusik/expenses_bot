package bot

import (
	"context"
	"expense-tracker/bot/internal/telegram"
	"log"
	"strings"
	"time"
)

const welcome = "Привет! Я веду учёт расходов. Просто напишите, что потратили: «кофе 350» или «такси 1200 вчера».\n" +
	"Можно спросить «сколько потратил на еду в этом месяце», поправить («последнее было не 350, а 450») или удалить («удали такси»)."

func (b *Bot) handle(ctx context.Context, m *telegram.Message) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	userID := m.From.ID
	if len(b.allowed) > 0 && !b.allowed[userID] {
		_ = b.tg.SendMessage(ctx, m.Chat.ID, "Нет доступа.")
		return
	}
	text := strings.TrimSpace(m.Text)
	if text == "/start" {
		_ = b.tg.SendMessage(ctx, m.Chat.ID, welcome)
		return
	}

	if err := b.api.UpsertUser(ctx, userID, m.From.DisplayName()); err != nil {
		log.Printf("upsert user %d: %v", userID, err)
	}
	b.tg.SendTyping(ctx, m.Chat.ID)

	reply, err := b.agent.Handle(ctx, userID, text)
	if err != nil {
		kind, userMsg := classify(err)
		log.Printf("%s (user %d): %v", kind, userID, err)
		reply = userMsg
	}
	if err := b.tg.SendMessage(ctx, m.Chat.ID, reply); err != nil {
		log.Printf("send (user %d): %v", userID, err)
	}
}
