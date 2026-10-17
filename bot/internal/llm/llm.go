// Package llm — минимальный клиент OpenAI-совместимого Chat Completions API с tool calling.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	http    *http.Client
	baseURL string
	apiKey  string
	model   string
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
		return Message{}, fmt.Errorf("llm: status %d: %s", resp.StatusCode, truncate(string(raw), 500))
	}

	var out struct {
		Choices []struct {
			Message Message `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return Message{}, fmt.Errorf("llm: bad response: %w", err)
	}
	if len(out.Choices) == 0 {
		return Message{}, fmt.Errorf("llm: empty choices")
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
