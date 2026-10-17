CREATE TABLE IF NOT EXISTS users (
    id         BIGINT PRIMARY KEY,           -- Telegram ID
    name       TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS expenses (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL,              -- Telegram ID владельца
    amount     NUMERIC(12,2) NOT NULL CHECK (amount > 0),
    category   TEXT NOT NULL CHECK (category IN
               ('еда','транспорт','жильё','развлечения','здоровье','покупки','другое')),
    spent_at   DATE NOT NULL,
    comment    TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_expenses_user_date ON expenses (user_id, spent_at DESC);
CREATE INDEX IF NOT EXISTS idx_expenses_date ON expenses (spent_at DESC);
