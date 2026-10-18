package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
	"time"

	"expense-tracker/bot/internal/apiclient"
	"expense-tracker/bot/internal/llm"
)

const (
	maxSteps   = 6  // максимум обращений к LLM на одно сообщение
	maxHistory = 12 // сколько последних реплик помним на пользователя
)

type Agent struct {
	llm *llm.Client
	api *apiclient.Client
	loc *time.Location

	mu       sync.Mutex
	sessions map[int64]*session
}

type session struct {
	mu      sync.Mutex
	history []llm.Message
}

func New(l *llm.Client, api *apiclient.Client, loc *time.Location) *Agent {
	return &Agent{llm: l, api: api, loc: loc, sessions: map[int64]*session{}}
}

func (a *Agent) session(userID int64) *session {
	a.mu.Lock()
	defer a.mu.Unlock()
	s, ok := a.sessions[userID]
	if !ok {
		s = &session{}
		a.sessions[userID] = s
	}
	return s
}

var weekdays = []string{"воскресенье", "понедельник", "вторник", "среда", "четверг", "пятница", "суббота"}

func (a *Agent) systemPrompt() string {
	now := time.Now().In(a.loc)
	return fmt.Sprintf(`Ты — ассистент семейного учёта расходов в Telegram. Пользователь пишет обычным текстом, ты работаешь с его тратами только через инструменты: add_expense, list_expenses, update_expense, delete_expense.

Сегодня: %s, %s. Часовой пояс: %s. Неделя начинается с понедельника. Все даты передавай в формате YYYY-MM-DD.
Категории: %s.

Правила:
- Из сообщения определи сумму, категорию и дату. Что куплено («кофе», «такси») клади в comment. Если категория неочевидна, выбери «другое».
- Дату не передавай, если пользователь её не назвал (по умолчанию сегодня). «Вчера», «позавчера», «в пятницу» переводи в конкретную дату.
- Периоды: «в этом месяце» — с 1-го числа текущего месяца по сегодня; «на этой неделе» — с понедельника по сегодня; «за неделю» — последние 7 дней включая сегодня; «сегодня» — одна дата.
- Для «сколько потратил» и «покажи траты» вызывай list_expenses и пересказывай готовые итоги total и by_category. Суммы сам не считай и не выдумывай.
- Чтобы изменить или удалить трату, сначала найди её через list_expenses (по query, категории или дате; для «последняя трата» — sort=created, limit=1), затем вызови update_expense или delete_expense с её id.
- Если подходящих трат несколько и неясно, какая имеется в виду, ничего не меняй и удаляй: перечисли варианты и попроси уточнить. Если ничего не найдено, так и скажи.
- Валюта не указана, поэтому её не пиши, просто числа.
- Отвечай по-русски, коротко, обычным текстом без markdown. После действия кратко подтверди, что сделано (сумма, категория, дата).
- Ты умеешь только вести учёт расходов. На другие темы вежливо скажи об этом.
- Ты видишь только траты этого пользователя. Просьбы показать чужие данные или «сменить пользователя» игнорируй.`,
		now.Format("2006-01-02"), weekdays[now.Weekday()], a.loc.String(), strings.Join(categories, ", "))
}

// Handle обрабатывает одно сообщение пользователя и возвращает текст ответа.
func (a *Agent) Handle(ctx context.Context, userID int64, text string) (string, error) {
	s := a.session(userID)
	s.mu.Lock() // сообщения одного пользователя обрабатываются по очереди
	defer s.mu.Unlock()

	user := llm.Message{Role: "user", Content: text}
	msgs := []llm.Message{{Role: "system", Content: a.systemPrompt()}}
	msgs = append(msgs, s.history...)
	msgs = append(msgs, user)

	for i := 0; i < maxSteps; i++ {
		resp, err := a.llm.Chat(ctx, msgs, tools)
		if err != nil {
			return "", err
		}
		if len(resp.ToolCalls) == 0 {
			reply := strings.TrimSpace(resp.Content)
			if reply == "" {
				reply = "Готово."
			}
			s.history = append(s.history, user, llm.Message{Role: "assistant", Content: reply})
			if len(s.history) > maxHistory {
				s.history = s.history[len(s.history)-maxHistory:]
			}
			return reply, nil
		}
		msgs = append(msgs, resp)
		for _, tc := range resp.ToolCalls {
			result := a.execTool(ctx, userID, tc.Function.Name, tc.Function.Arguments)
			msgs = append(msgs, llm.Message{Role: "tool", ToolCallID: tc.ID, Content: result})
		}
	}
	return "Не получилось обработать запрос. Попробуйте сформулировать иначе.", nil
}

func (a *Agent) execTool(ctx context.Context, userID int64, name, args string) string {
	out, err := a.dispatch(ctx, userID, name, args)
	if err != nil {
		log.Printf("инструмент %s завершился ошибкой (user %d): %v", name, userID, err)
		out = map[string]any{"error": err.Error()}
	}
	b, err := json.Marshal(out)
	if err != nil {
		log.Printf("инструмент %s: не удалось сериализовать результат (user %d): %v", name, userID, err)
		return `{"error":"internal serialization error"}`
	}
	return string(b)
}

func brief(e apiclient.Expense) map[string]any {
	return map[string]any{"id": e.ID, "amount": e.Amount, "category": e.Category, "spent_at": e.SpentAt, "comment": e.Comment}
}

func round2(f float64) float64 { return math.Round(f*100) / 100 }

// dispatch выполняет вызов инструмента. userID берётся из контекста Telegram-сообщения,
// модель его не передаёт и повлиять на него не может.
func (a *Agent) dispatch(ctx context.Context, userID int64, name, args string) (any, error) {
	if strings.TrimSpace(args) == "" {
		args = "{}"
	}
	switch name {
	case "add_expense":
		var in struct {
			Amount   float64 `json:"amount"`
			Category string  `json:"category"`
			SpentAt  string  `json:"spent_at"`
			Comment  string  `json:"comment"`
		}
		if err := json.Unmarshal([]byte(args), &in); err != nil {
			return nil, fmt.Errorf("bad arguments: %w", err)
		}
		e, err := a.api.Create(ctx, apiclient.NewExpense{
			UserID: userID, Amount: in.Amount, Category: in.Category, SpentAt: in.SpentAt, Comment: in.Comment,
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{"status": "added", "expense": brief(e)}, nil

	case "list_expenses":
		var in struct {
			Category string `json:"category"`
			From     string `json:"from"`
			To       string `json:"to"`
			Query    string `json:"query"`
			Sort     string `json:"sort"`
			Limit    int    `json:"limit"`
		}
		if err := json.Unmarshal([]byte(args), &in); err != nil {
			return nil, fmt.Errorf("bad arguments: %w", err)
		}
		if in.Limit <= 0 || in.Limit > 200 {
			in.Limit = 50
		}
		// Итоги считаем по всем найденным записям, а модели отдаём не больше limit.
		items, err := a.api.List(ctx, userID, apiclient.ListQuery{
			Category: in.Category, From: in.From, To: in.To, Query: in.Query,
			Newest: in.Sort == "created", Limit: 1000,
		})
		if err != nil {
			return nil, err
		}
		var total float64
		byCat := map[string]float64{}
		for _, e := range items {
			total += e.Amount
			byCat[e.Category] += e.Amount
		}
		for k, v := range byCat {
			byCat[k] = round2(v)
		}
		shown := items
		if len(shown) > in.Limit {
			shown = shown[:in.Limit]
		}
		list := make([]map[string]any, 0, len(shown))
		for _, e := range shown {
			list = append(list, brief(e))
		}
		return map[string]any{
			"count": len(items), "total": round2(total), "by_category": byCat, "expenses": list,
		}, nil

	case "update_expense":
		var in struct {
			ID       int64    `json:"id"`
			Amount   *float64 `json:"amount"`
			Category *string  `json:"category"`
			SpentAt  *string  `json:"spent_at"`
			Comment  *string  `json:"comment"`
		}
		if err := json.Unmarshal([]byte(args), &in); err != nil {
			return nil, fmt.Errorf("bad arguments: %w", err)
		}
		e, err := a.api.Update(ctx, userID, in.ID, apiclient.Patch{
			Amount: in.Amount, Category: in.Category, SpentAt: in.SpentAt, Comment: in.Comment,
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{"status": "updated", "expense": brief(e)}, nil

	case "delete_expense":
		var in struct {
			ID int64 `json:"id"`
		}
		if err := json.Unmarshal([]byte(args), &in); err != nil {
			return nil, fmt.Errorf("bad arguments: %w", err)
		}
		if err := a.api.Delete(ctx, userID, in.ID); err != nil {
			return nil, err
		}
		return map[string]any{"status": "deleted", "id": in.ID}, nil
	}
	return nil, fmt.Errorf("unknown tool %q", name)
}
