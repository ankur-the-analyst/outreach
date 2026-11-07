package ai

import (
	"context"
	"encoding/json"
	"fmt"
)

type Anthropic struct {
	Model string
}

func (Anthropic) Complete(context.Context, Request) (TextResult, error) {
	return TextResult{}, ErrNotConfigured
}

func (Anthropic) CompleteJSON(context.Context, Request, json.RawMessage) (JSONResult, error) {
	return JSONResult{}, ErrNotConfigured
}

func (Anthropic) Embed(context.Context, []string) ([][]float32, error) {
	return nil, fmt.Errorf("%w", ErrAnthropicEmbed)
}
