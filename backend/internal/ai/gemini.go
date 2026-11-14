package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"google.golang.org/genai"
)

const geminiDefaultModel = "gemini-3.8-flash"

type Gemini struct {
	Model  string
	client *genai.Client
}

func newGemini(b Binding) (*Gemini, error) {
	cfg := &genai.ClientConfig{
		APIKey:     b.APIKey,
		Backend:    genai.BackendGeminiAPI,
		HTTPClient: b.HTTP,
	}
	if b.HTTP == nil {
		cfg.HTTPClient = &http.Client{}
	}
	if base := strings.TrimSpace(b.BaseURL); base != "" {
		cfg.HTTPOptions.BaseURL = base
	}
	client, err := genai.NewClient(context.Background(), cfg)
	if err != nil {
		return nil, fmt.Errorf("ai: gemini: %w", err)
	}
	return &Gemini{Model: b.Model, client: client}, nil
}

func (g *Gemini) Complete(ctx context.Context, req Request) (TextResult, error) {
	text, model, err := g.generate(ctx, req, false)
	if err != nil {
		return TextResult{}, err
	}
	return TextResult{Text: text, Model: model}, nil
}

func (g *Gemini) CompleteJSON(ctx context.Context, req Request, _ json.RawMessage) (JSONResult, error) {
	text, model, err := g.generate(ctx, req, true)
	if err != nil {
		return JSONResult{}, err
	}
	raw := json.RawMessage(strings.TrimSpace(text))
	if !json.Valid(raw) {
		return JSONResult{}, fmt.Errorf("ai: gemini json: %s", text)
	}
	return JSONResult{Raw: raw, Model: model}, nil
}

func (g *Gemini) Embed(context.Context, []string) ([][]float32, error) {
	return nil, ErrNotSupported
}

func (g *Gemini) generate(ctx context.Context, req Request, asJSON bool) (string, string, error) {
	model := strings.TrimSpace(g.Model)
	if model == "" {
		model = geminiDefaultModel
	}
	temp := float32(req.Temperature)
	cfg := &genai.GenerateContentConfig{Temperature: &temp}
	if req.System != "" {
		cfg.SystemInstruction = &genai.Content{Parts: []*genai.Part{{Text: req.System}}}
	}
	if asJSON {
		cfg.ResponseMIMEType = "application/json"
	}
	res, err := g.client.Models.GenerateContent(ctx, model, genai.Text(req.User), cfg)
	if err != nil {
		return "", "", fmt.Errorf("ai: gemini: %w", err)
	}
	text := strings.TrimSpace(res.Text())
	if text == "" {
		return "", "", fmt.Errorf("ai: gemini empty response")
	}
	return text, model, nil
}
