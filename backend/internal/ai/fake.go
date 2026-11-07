package ai

import (
	"context"
	"encoding/json"
	"fmt"
)

type Fake struct {
	Model     string
	Text      string
	JSON      json.RawMessage
	Vectors   [][]float32
	CompleteN int
	JSONN     int
	EmbedN    int
}

func (f *Fake) Complete(context.Context, Request) (TextResult, error) {
	f.CompleteN++
	model := f.Model
	if model == "" {
		model = "fake"
	}
	return TextResult{Text: f.Text, Model: model}, nil
}

func (f *Fake) CompleteJSON(context.Context, Request, json.RawMessage) (JSONResult, error) {
	f.JSONN++
	model := f.Model
	if model == "" {
		model = "fake"
	}
	raw := f.JSON
	if len(raw) == 0 {
		raw = json.RawMessage(`{}`)
	}
	return JSONResult{Raw: raw, Model: model}, nil
}

func (f *Fake) Embed(_ context.Context, texts []string) ([][]float32, error) {
	f.EmbedN++
	if f.Vectors != nil {
		if len(f.Vectors) != len(texts) {
			return nil, fmt.Errorf("ai: fake embed len %d want %d", len(f.Vectors), len(texts))
		}
		return f.Vectors, nil
	}
	out := make([][]float32, len(texts))
	for i := range texts {
		out[i] = []float32{float32(len(texts[i]))}
	}
	return out, nil
}
