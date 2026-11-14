package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const openaiDefaultBase = "https://api.openai.com/v1"

type OpenAI struct {
	Model   string
	APIKey  string
	BaseURL string
	HTTP    *http.Client
}

func (o *OpenAI) Complete(ctx context.Context, req Request) (TextResult, error) {
	text, model, err := o.chat(ctx, req, false)
	if err != nil {
		return TextResult{}, err
	}
	return TextResult{Text: text, Model: model}, nil
}

func (o *OpenAI) CompleteJSON(ctx context.Context, req Request, _ json.RawMessage) (JSONResult, error) {
	text, model, err := o.chat(ctx, req, true)
	if err != nil {
		return JSONResult{}, err
	}
	raw := json.RawMessage(strings.TrimSpace(text))
	if !json.Valid(raw) {
		return JSONResult{}, fmt.Errorf("ai: openai json: %s", text)
	}
	return JSONResult{Raw: raw, Model: model}, nil
}

func (o *OpenAI) Embed(context.Context, []string) ([][]float32, error) {
	return nil, ErrNotSupported
}

func (o *OpenAI) chat(ctx context.Context, req Request, asJSON bool) (string, string, error) {
	base := strings.TrimRight(strings.TrimSpace(o.BaseURL), "/")
	if base == "" {
		base = openaiDefaultBase
	}
	model := o.Model
	if model == "" {
		model = "gpt-6-luna"
	}
	payload := map[string]any{
		"model":       model,
		"temperature": req.Temperature,
		"messages": []map[string]string{
			{"role": "system", "content": req.System},
			{"role": "user", "content": req.User},
		},
	}
	if asJSON {
		payload["response_format"] = map[string]string{"type": "json_object"}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", "", err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	httpReq.Header.Set("Authorization", "Bearer "+o.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	cli := o.HTTP
	if cli == nil {
		cli = &http.Client{Timeout: 30 * time.Second}
	}
	res, err := cli.Do(httpReq)
	if err != nil {
		return "", "", fmt.Errorf("ai: openai: %w", err)
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return "", "", err
	}
	if res.StatusCode >= 300 {
		return "", "", fmt.Errorf("ai: openai: http %d", res.StatusCode)
	}
	var out struct {
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", "", fmt.Errorf("ai: openai decode: %w", err)
	}
	if len(out.Choices) == 0 {
		return "", "", fmt.Errorf("ai: openai empty choices")
	}
	if out.Model == "" {
		out.Model = model
	}
	return strings.TrimSpace(out.Choices[0].Message.Content), out.Model, nil
}
