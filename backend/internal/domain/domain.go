package domain

import (
	"errors"
	"math"
	"strings"
	"time"
)

const DateLayout = "2006-01-02"

var Categories = []string{"еда", "транспорт", "жильё", "развлечения", "здоровье", "покупки", "другое"}

// Expense — трата. SpentAt хранится как дата в формате YYYY-MM-DD.
type Expense struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Amount    float64   `json:"amount"`
	Category  string    `json:"category"`
	SpentAt   string    `json:"spent_at"`
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type User struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Filter struct {
	UserID         *int64
	Category       string
	From, To       string // YYYY-MM-DD, включительно
	Query          string // подстрока в комментарии
	OrderByCreated bool   // сортировать по времени создания, а не по дате траты
	Limit          int
}

// Patch — частичное обновление: nil означает «не менять».
type Patch struct {
	Amount   *float64
	Category *string
	SpentAt  *string
	Comment  *string
}

var ErrNotFound = errors.New("expense not found")

type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

func bad(msg string) error { return &ValidationError{Msg: msg} }

// ---- валидация и нормализация ----

func NormAmount(a float64) (float64, error) {
	if math.IsNaN(a) || a <= 0 {
		return 0, bad("amount must be greater than 0")
	}
	if a >= 1e10 {
		return 0, bad("amount is too large")
	}
	return math.Round(a*100) / 100, nil
}

func NormCategory(c string) (string, error) {
	c = strings.ToLower(strings.TrimSpace(c))
	if c == "" {
		return "другое", nil
	}
	if c == "жилье" {
		c = "жильё"
	}
	for _, x := range Categories {
		if x == c {
			return c, nil
		}
	}
	return "", bad("unknown category, allowed: " + strings.Join(Categories, ", "))
}

func NormDate(d string) (string, error) {
	t, err := time.Parse(DateLayout, strings.TrimSpace(d))
	if err != nil {
		return "", bad("date must be in YYYY-MM-DD format")
	}
	return t.Format(DateLayout), nil
}

// Normalize проверяет новую трату. Пустая дата заменяется на today.
func (e *Expense) Normalize(today string) (err error) {
	if e.UserID == 0 {
		return bad("user_id is required")
	}
	if e.Amount, err = NormAmount(e.Amount); err != nil {
		return err
	}
	if e.Category, err = NormCategory(e.Category); err != nil {
		return err
	}
	if strings.TrimSpace(e.SpentAt) == "" {
		e.SpentAt = today
	} else if e.SpentAt, err = NormDate(e.SpentAt); err != nil {
		return err
	}
	e.Comment = strings.TrimSpace(e.Comment)
	return nil
}

func (p *Patch) Normalize() error {
	if p.Amount != nil {
		v, err := NormAmount(*p.Amount)
		if err != nil {
			return err
		}
		p.Amount = &v
	}
	if p.Category != nil {
		v, err := NormCategory(*p.Category)
		if err != nil {
			return err
		}
		p.Category = &v
	}
	if p.SpentAt != nil {
		v, err := NormDate(*p.SpentAt)
		if err != nil {
			return err
		}
		p.SpentAt = &v
	}
	if p.Comment != nil {
		v := strings.TrimSpace(*p.Comment)
		p.Comment = &v
	}
	return nil
}

func (f *Filter) Normalize() (err error) {
	if f.Category != "" {
		if f.Category, err = NormCategory(f.Category); err != nil {
			return err
		}
	}
	if f.From != "" {
		if f.From, err = NormDate(f.From); err != nil {
			return err
		}
	}
	if f.To != "" {
		if f.To, err = NormDate(f.To); err != nil {
			return err
		}
	}
	if f.Limit <= 0 {
		f.Limit = 500
	}
	if f.Limit > 1000 {
		f.Limit = 1000
	}
	return nil
}
