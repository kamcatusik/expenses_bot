// Package apiclient — клиент REST API бэкенда.
package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Client struct {
	base string
	http *http.Client
}

func NewClient(base string) *Client {
	return &Client{base: base, http: &http.Client{Timeout: 15 * time.Second}}
}

type Expense struct {
	ID       int64   `json:"id"`
	UserID   int64   `json:"user_id"`
	Amount   float64 `json:"amount"`
	Category string  `json:"category"`
	SpentAt  string  `json:"spent_at"`
	Comment  string  `json:"comment"`
}

type NewExpense struct {
	UserID   int64   `json:"user_id"`
	Amount   float64 `json:"amount"`
	Category string  `json:"category"`
	SpentAt  string  `json:"spent_at,omitempty"`
	Comment  string  `json:"comment,omitempty"`
}

type Patch struct {
	Amount   *float64 `json:"amount,omitempty"`
	Category *string  `json:"category,omitempty"`
	SpentAt  *string  `json:"spent_at,omitempty"`
	Comment  *string  `json:"comment,omitempty"`
}

type ListQuery struct {
	Category string
	From, To string
	Query    string
	Newest   bool // сортировка по времени добавления
	Limit    int
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var rd *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(b)
	} else {
		rd = bytes.NewReader(nil)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, rd)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&e)
		if e.Error == "" {
			e.Error = resp.Status
		}
		return fmt.Errorf("%s", e.Error)
	}
	if out != nil && resp.StatusCode != http.StatusNoContent {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func owner(userID int64) string { return "user_id=" + strconv.FormatInt(userID, 10) }

func (c *Client) Create(ctx context.Context, e NewExpense) (Expense, error) {
	var out Expense
	err := c.do(ctx, http.MethodPost, "/api/expenses", e, &out)
	return out, err
}

// List всегда ограничен userID — бот не умеет читать чужие траты.
func (c *Client) List(ctx context.Context, userID int64, q ListQuery) ([]Expense, error) {
	v := url.Values{}
	v.Set("user_id", strconv.FormatInt(userID, 10))
	if q.Category != "" {
		v.Set("category", q.Category)
	}
	if q.From != "" {
		v.Set("from", q.From)
	}
	if q.To != "" {
		v.Set("to", q.To)
	}
	if q.Query != "" {
		v.Set("q", q.Query)
	}
	if q.Newest {
		v.Set("order", "created")
	}
	if q.Limit > 0 {
		v.Set("limit", strconv.Itoa(q.Limit))
	}
	var out []Expense
	err := c.do(ctx, http.MethodGet, "/api/expenses?"+v.Encode(), nil, &out)
	return out, err
}

func (c *Client) Update(ctx context.Context, userID, id int64, p Patch) (Expense, error) {
	var out Expense
	err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/expenses/%d?%s", id, owner(userID)), p, &out)
	return out, err
}

func (c *Client) Delete(ctx context.Context, userID, id int64) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/expenses/%d?%s", id, owner(userID)), nil, nil)
}

func (c *Client) UpsertUser(ctx context.Context, id int64, name string) error {
	return c.do(ctx, http.MethodPost, "/api/users", map[string]any{"id": id, "name": name}, nil)
}
