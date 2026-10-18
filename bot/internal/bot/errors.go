package bot

import (
	"errors"
	"expense-tracker/bot/internal/llm"
)

func classify(err error) (kind, userMsg string) {
	var se *llm.StatusError
	switch {
	case errors.As(err, &se) && se.Code == 429:
		return "LLM: превышен лимит запросов (429)", "ИИ-сервис сейчас перегружен. Повторите сообщение через минуту."
	case errors.As(err, &se) && (se.Code == 401 || se.Code == 403):
		return "LLM: ключ отклонён, проверьте LLM_API_KEY", "ИИ-сервис временно недоступен. Попробуйте позже."
	case errors.Is(err, llm.ErrLLM):
		return "LLM: сервис недоступен или вернул ошибку", "ИИ-сервис временно недоступен. Попробуйте позже."
	default:
		return "ошибка обработки сообщения", "Что-то пошло не так. Попробуйте ещё раз чуть позже."
	}
}
