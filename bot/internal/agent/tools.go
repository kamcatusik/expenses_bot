package agent

import "expense-tracker/bot/internal/llm"

var categories = []string{"еда", "транспорт", "жильё", "развлечения", "здоровье", "покупки", "другое"}

func obj(props map[string]any, required ...string) map[string]any {
	if required == nil {
		required = []string{}
	}
	return map[string]any{"type": "object", "properties": props, "required": required}
}

func str(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }
func num(desc string) map[string]any { return map[string]any{"type": "number", "description": desc} }
func integer(desc string) map[string]any {
	return map[string]any{"type": "integer", "description": desc}
}

func catProp() map[string]any {
	return map[string]any{"type": "string", "enum": categories, "description": "Категория траты"}
}

// Обратите внимание: user_id в схемах нет. Его подставляет код бота из Telegram-сообщения.
var tools = []llm.Tool{
	{Type: "function", Function: llm.ToolFunc{
		Name:        "add_expense",
		Description: "Добавить трату текущего пользователя.",
		Parameters: obj(map[string]any{
			"amount":   num("Сумма, больше 0"),
			"category": catProp(),
			"spent_at": str("Дата траты YYYY-MM-DD. Не передавай, если пользователь не назвал дату (будет сегодня)."),
			"comment":  str("Что куплено, коротко: «кофе», «такси до дома»"),
		}, "amount", "category"),
	}},
	{Type: "function", Function: llm.ToolFunc{
		Name: "list_expenses",
		Description: "Получить траты текущего пользователя с фильтрами. Возвращает список, " +
			"а также готовые итоги (total, by_category) — суммы сам не считай.",
		Parameters: obj(map[string]any{
			"category": catProp(),
			"from":     str("Начало периода YYYY-MM-DD, включительно"),
			"to":       str("Конец периода YYYY-MM-DD, включительно"),
			"query":    str("Подстрока для поиска в комментарии, например «такси»"),
			"sort": map[string]any{
				"type": "string", "enum": []string{"date", "created"},
				"description": "date — по дате траты (по умолчанию); created — по времени добавления, нужно для «последняя трата»",
			},
			"limit": integer("Максимум записей, по умолчанию 50"),
		}),
	}},
	{Type: "function", Function: llm.ToolFunc{
		Name:        "update_expense",
		Description: "Изменить трату по id. Передавай только изменяемые поля.",
		Parameters: obj(map[string]any{
			"id":       integer("id траты из list_expenses"),
			"amount":   num("Новая сумма"),
			"category": catProp(),
			"spent_at": str("Новая дата YYYY-MM-DD"),
			"comment":  str("Новый комментарий"),
		}, "id"),
	}},
	{Type: "function", Function: llm.ToolFunc{
		Name:        "delete_expense",
		Description: "Удалить трату по id.",
		Parameters:  obj(map[string]any{"id": integer("id траты из list_expenses")}, "id"),
	}},
}
