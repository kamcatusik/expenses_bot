# Семейные расходы: Telegram-бот с ИИ + веб-панель

Член семьи пишет боту «кофе 350», и трата сохраняется: сумму, категорию и дату определяет LLM.
Глава семьи видит и правит траты всех в веб-панели.

## Запуск

```bash
cp .env.example .env     # впишите TELEGRAM_TOKEN, LLM_API_KEY, при необходимости LLM_BASE_URL / LLM_MODEL / TZ
docker compose up --build
```

- Веб-панель: http://localhost:3000
- REST API (через nginx): http://localhost:3000/api/expenses

Один `docker compose` поднимает четыре контейнера: `db` (Postgres), `backend`, `bot`, `frontend`.
Миграции применяются автоматически при старте backend.

## Архитектура

```
Telegram ──► bot (Go) ──REST──► backend (Go) ──► Postgres
                │                    ▲
                └──► LLM API         │ REST
                                  frontend (nginx + React SPA)
```

| Папка | Что внутри |
|---|---|
| `backend/` | REST API и работа с БД. Единственный владелец данных. |
| `bot/` | Telegram long polling, ИИ-агент (function calling), клиент к backend. Без внешних зависимостей. |
| `frontend/` | React + Vite, раздаётся nginx, `/api` проксируется на backend. |

Backend без отдельного слоя сервиса: валидация и нормализация лежат в `domain`,
хендлеры работают с интерфейсами `repository.Expenses` / `repository.Users`.

```
backend/internal/
├── domain/      сущности, ошибки, валидация (Normalize)
├── repository/  интерфейсы Expenses/Users и реализация на Postgres
└── transport/http/  роутер и хендлеры
```

## Настройка LLM

Подходит любой OpenAI-совместимый API с function calling (OpenAI, OpenRouter, Ollama, vLLM и т. д.):

```
LLM_BASE_URL=https://api.openai.com/v1
LLM_API_KEY=...
LLM_MODEL=gpt-4o-mini
```

## Как работает бот

У агента четыре инструмента: `add_expense`, `list_expenses`, `update_expense`, `delete_expense`.

- **Изоляция данных.** В схемах инструментов нет `user_id`. Его подставляет код бота из Telegram-сообщения,
  модель его не видит и не может изменить. Дополнительно backend сам проверяет владельца в SQL
  (`WHERE id = $1 AND user_id = $2`): чужая трата выглядит как несуществующая.
- **Даты.** В system prompt передаются текущая дата, день недели и часовой пояс (`TZ`),
  поэтому «вчера», «на этой неделе», «в этом месяце» считаются правильно.
- **Итоги.** Суммы и разбивку по категориям считает код (`total`, `by_category`), а не модель.
- **Изменить/удалить.** Агент сначала находит трату через `list_expenses`
  («последняя» = сортировка по времени добавления), потом вызывает `update_expense`/`delete_expense`.
  Если под описание подходит несколько трат, бот переспрашивает.
- Бот помнит последние ~12 реплик на пользователя (в памяти процесса), поэтому работают уточнения.
- `ALLOWED_USER_IDS` (необязательно) ограничивает бота списком Telegram ID.

## REST API

| Метод | Путь | Назначение |
|---|---|---|
| GET | `/api/expenses?user_id=&category=&from=&to=&q=&order=&limit=` | список (`order=created` сортирует по времени добавления) |
| POST | `/api/expenses` | создать: `{user_id, amount, category, spent_at?, comment?}` |
| GET | `/api/expenses/{id}[?user_id=]` | получить |
| PATCH | `/api/expenses/{id}[?user_id=]` | изменить (любые из `amount, category, spent_at, comment`) |
| DELETE | `/api/expenses/{id}[?user_id=]` | удалить |
| GET / POST | `/api/users` | члены семьи; бот регистрирует пользователя при первом сообщении |
| GET | `/api/categories` | список категорий |

Если передан `user_id`, доступ ограничен тратами этого пользователя (так бот ходит в API).
Веб-панель его не передаёт в запросах по id и работает со всеми тратами. Авторизации нет по условию задания.

Пример:

```bash
curl -X POST localhost:3000/api/expenses -H 'Content-Type: application/json' \
  -d '{"user_id":1,"amount":350,"category":"еда","comment":"кофе"}'
```

## Локальная разработка

```bash
docker compose up db -d
cd backend && DATABASE_URL='postgres://app:app@localhost:5432/expenses?sslmode=disable' go run ./cmd/api
cd frontend && npm install && npm run dev   # проксирует /api на localhost:8080
```

На Docker-сборке `go.sum` для backend создаётся командой `go mod tidy`, поэтому нужен доступ в интернет.
