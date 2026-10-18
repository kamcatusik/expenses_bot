// Package llm — минимальный клиент OpenAI-совместимого Chat Completions API с tool calling.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const maxAttempts = 3

type Client struct {
	http    *http.Client
	baseURL string
	apiKey  string
	model   string
}

// ErrLLM помечает любую ошибку, источник которой — сама нейросеть/её API
// (сеть, таймаут, статус, битый ответ). Проверяется через errors.Is.
var ErrLLM = errors.New("проблема с LLM")

type StatusError struct {
	Code int
	Body string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("status %d: %s", e.Code, truncate(e.Body, 500))
}
func NewClient(baseURL, apiKey, model string) *Client {
	return &Client{baseURL: baseURL, apiKey: apiKey, model: model, http: &http.Client{Timeout: 60 * time.Second}}
}

type FuncCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ToolCall struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Function FuncCall `json:"function"`
}

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type ToolFunc struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type Tool struct {
	Type     string   `json:"type"`
	Function ToolFunc `json:"function"`
}

func (c *Client) Chat(ctx context.Context, msgs []Message, tools []Tool) (Message, error) {
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-time.After(time.Duration(attempt) * 2 * time.Second):
			case <-ctx.Done():
				return Message{}, fmt.Errorf("%w: %w", ErrLLM, ctx.Err())
			}
		}
		m, err := c.chatOnce(ctx, msgs, tools)
		if err == nil {
			return m, nil
		}
		lastErr = err
		var se *StatusError
		if !errors.As(err, &se) || (se.Code != 429 && se.Code < 500) {
			break // повторять имеет смысл только при перегрузке и сбоях сервера
		}
	}
	return Message{}, fmt.Errorf("%w: %w", ErrLLM, lastErr)
}
func (c *Client) chatOnce(ctx context.Context, msgs []Message, tools []Tool) (Message, error) {
	body, err := json.Marshal(map[string]any{
		"model":       c.model,
		"messages":    msgs,
		"tools":       tools,
		"tool_choice": "auto",
	})
	if err != nil {
		return Message{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Message{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return Message{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return Message{}, &StatusError{Code: resp.StatusCode, Body: string(raw)}
	}

	var out struct {
		Choices []struct {
			Message Message `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return Message{}, fmt.Errorf("bad response: %w", err)
	}
	if len(out.Choices) == 0 {
		return Message{}, errors.New("empty choices")
	}
	m := out.Choices[0].Message
	m.Role = "assistant"
	return m, nil
}
func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}
