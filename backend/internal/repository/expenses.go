package repository

import (
	"context"
	"database/sql"
	"errors"

	"expense-tracker/backend/internal/domain"
)

// Expenses — хранилище трат. owner != nil ограничивает доступ тратами одного пользователя;
// чужая трата для такого вызова неотличима от несуществующей (ErrNotFound).
type Expenses interface {
	Create(ctx context.Context, e domain.Expense) (domain.Expense, error)
	Get(ctx context.Context, id int64, owner *int64) (domain.Expense, error)
	List(ctx context.Context, f domain.Filter) ([]domain.Expense, error)
	Update(ctx context.Context, id int64, owner *int64, p domain.Patch) (domain.Expense, error)
	Delete(ctx context.Context, id int64, owner *int64) error
}

const cols = `id, user_id, amount::float8, category, to_char(spent_at, 'YYYY-MM-DD'), comment, created_at, updated_at`

type expenses struct{ db *sql.DB }
type scanner interface{ Scan(dest ...any) error }

func NewExpenses(db *sql.DB) Expenses { return &expenses{db: db} }

func scanExpense(s scanner) (domain.Expense, error) {
	var e domain.Expense
	err := s.Scan(&e.ID, &e.UserID, &e.Amount, &e.Category, &e.SpentAt, &e.Comment, &e.CreatedAt, &e.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return e, domain.ErrNotFound
	}
	return e, err
}

func (r *expenses) Create(ctx context.Context, e domain.Expense) (domain.Expense, error) {
	return scanExpense(r.db.QueryRowContext(ctx,
		`INSERT INTO expenses (user_id, amount, category, spent_at, comment)
		 VALUES ($1, $2, $3, $4, $5) RETURNING `+cols,
		e.UserID, e.Amount, e.Category, date(e.SpentAt), e.Comment))
}

func (r *expenses) Get(ctx context.Context, id int64, owner *int64) (domain.Expense, error) {
	return scanExpense(r.db.QueryRowContext(ctx,
		`SELECT `+cols+` FROM expenses WHERE id = $1 AND ($2::bigint IS NULL OR user_id = $2)`, id, owner))
}

func (r *expenses) List(ctx context.Context, f domain.Filter) ([]domain.Expense, error) {
	var (
		orderBySpent   = ` ORDER BY spent_at DESC, id DESC LIMIT $6`
		orderByCreated = ` ORDER BY created_at DESC, id DESC LIMIT $6`
	)
	listBase := `SELECT ` + cols + ` FROM expenses
  WHERE ($1::bigint IS NULL OR user_id = $1)
  AND ($2::text   IS NULL OR category = $2)
  AND ($3::date   IS NULL OR spent_at >= $3)
  AND ($4::date   IS NULL OR spent_at <= $4)
  AND ($5::text   IS NULL OR comment ILIKE '%' || $5 || '%')`
	q := listBase + orderBySpent
	if f.OrderByCreated {
		q = listBase + orderByCreated
	}

	rows, err := r.db.QueryContext(ctx, q,
		f.UserID, nilIfEmpty(f.Category), dateOrNil(f.From), dateOrNil(f.To), nilIfEmpty(f.Query), f.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []domain.Expense{}
	for rows.Next() {
		e, err := scanExpense(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// Update меняет только переданные поля (COALESCE), проверка владельца идёт в том же запросе.
func (r *expenses) Update(ctx context.Context, id int64, owner *int64, p domain.Patch) (domain.Expense, error) {
	return scanExpense(r.db.QueryRowContext(ctx,
		`UPDATE expenses SET
		    amount     = COALESCE($3, amount),
		    category   = COALESCE($4, category),
		    spent_at   = COALESCE($5, spent_at),
		    comment    = COALESCE($6, comment),
		    updated_at = now()
		 WHERE id = $1 AND ($2::bigint IS NULL OR user_id = $2)
		 RETURNING `+cols,
		id, owner, p.Amount, p.Category, datePtr(p.SpentAt), p.Comment))
}

func (r *expenses) Delete(ctx context.Context, id int64, owner *int64) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM expenses WHERE id = $1 AND ($2::bigint IS NULL OR user_id = $2)`, id, owner)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrNotFound
	}
	return nil
}
