package repository

import (
	"context"
	"database/sql"

	"expense-tracker/backend/internal/domain"
)

type Users interface {
	Upsert(ctx context.Context, u domain.User) error
	// List возвращает  пользователей и тех, у кого есть траты, но нет записи в users.
	List(ctx context.Context) ([]domain.User, error)
}

type users struct{ db *sql.DB }

func NewUsers(db *sql.DB) Users { return &users{db: db} }

func (r *users) Upsert(ctx context.Context, u domain.User) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, name) VALUES ($1, $2)
		 ON CONFLICT (id) DO UPDATE
		 SET name = CASE WHEN EXCLUDED.name <> '' THEN EXCLUDED.name ELSE users.name END`,
		u.ID, u.Name)
	return err
}

func (r *users) List(ctx context.Context) ([]domain.User, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name FROM users
		 UNION
		 SELECT DISTINCT user_id, '' FROM expenses WHERE user_id NOT IN (SELECT id FROM users)
		 ORDER BY 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.User{}
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Name); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}
