CREATE TABLE IF NOT EXISTS users (
    id         BIGINT PRIMARY KEY,           -- Telegram ID
    name       TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS expenses (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id), -- Telegram ID владельца
    amount     NUMERIC(12,2) NOT NULL CHECK (amount > 0),
    category   TEXT NOT NULL CHECK (category IN
               ('еда','транспорт','жильё','развлечения','здоровье','покупки','другое')),
    spent_at   DATE NOT NULL,
    comment    TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);


