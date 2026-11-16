package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

var (
	ErrNotSupported   = errors.New("ai: not supported")
	ErrNotConfigured  = errors.New("ai: not configured")
	ErrNoBinding      = errors.New("ai: no binding for role")
	ErrNoKey          = errors.New("ai: api key required")
	ErrUnknownKind    = errors.New("ai: unknown provider kind")
	ErrAnthropicEmbed = errors.New("ai: embed cannot use anthropic")
	ErrUnknownModel   = errors.New("ai: unknown model")
)

type Role string

const (
	RoleCompose  Role = "compose"
	RoleAnalyze  Role = "analyze"
	RoleClassify Role = "classify"
	RoleReply    Role = "reply"
	RoleEmbed    Role = "embed"
)

type Kind string

const (
	KindOpenAI     Kind = "openai"
	KindGoogle     Kind = "google"
	KindAnthropic  Kind = "anthropic"
	KindOpenRouter Kind = "openrouter"
	KindCompat     Kind = "openai_compat"
	KindFake       Kind = "fake"
)

type Request struct {
	Role        Role
	System      string
	User        string
	Temperature float64
}

type TextResult struct {
	Text  string
	Model string
}

type JSONResult struct {
	Raw   json.RawMessage
	Model string
}

type Provider interface {
	Complete(ctx context.Context, req Request) (TextResult, error)
	CompleteJSON(ctx context.Context, req Request, schema json.RawMessage) (JSONResult, error)
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

type Binding struct {
	Role    Role
	Kind    Kind
	Model   string
	APIKey  string
	BaseURL string
	HTTP    *http.Client
}

func Temperature(r Role) float64 {
	switch r {
	case RoleAnalyze, RoleClassify:
		return 0.2
	default:
		return 0.6
	}
}

func New(b Binding) (Provider, error) {
	switch b.Kind {
	case KindFake:
		return &Fake{Model: b.Model}, nil
	case KindAnthropic:
		return Anthropic{Model: b.Model}, nil
	case KindOpenAI, KindCompat, KindOpenRouter:
		if b.APIKey == "" {
			return nil, ErrNoKey
		}
		base := b.BaseURL
		if b.Kind == KindOpenRouter && strings.TrimSpace(base) == "" {
			base = openRouterBase
		}
		return &OpenAI{Model: b.Model, APIKey: b.APIKey, BaseURL: base}, nil
	case KindGoogle:
		if b.APIKey == "" {
			return nil, ErrNoKey
		}
		return newGemini(b)
	default:
		return nil, ErrUnknownKind
	}
}
