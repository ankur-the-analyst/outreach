package ai

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = errors.New("not found")
	ErrKeyRequired  = errors.New("api key required")
	ErrBaseURL      = errors.New("base url required")
	ErrComposeModel = errors.New("compose model required")
	ErrEmbedModel   = errors.New("embed model required")
)

type Settings struct {
	Kind         Kind   `json:"kind,omitempty"`
	HasKey       bool   `json:"has_key"`
	Hint         string `json:"hint,omitempty"`
	BaseURL      string `json:"base_url,omitempty"`
	ComposeModel string `json:"compose_model,omitempty"`
	EmbedModel   string `json:"embed_model,omitempty"`
}

type PutInput struct {
	Kind         Kind
	APIKey       string
	BaseURL      string
	ComposeModel string
	EmbedModel   string
}

type storedProvider struct {
	ID       uuid.UUID `json:"id"`
	Kind     Kind      `json:"kind"`
	SecretID uuid.UUID `json:"secret_id"`
	BaseURL  string    `json:"base_url,omitempty"`
	Hint     string    `json:"hint"`
}

type storedFallback struct {
	Kind  Kind   `json:"kind"`
	Model string `json:"model"`
}

func kindOK(k Kind) bool {
	switch k {
	case KindOpenAI, KindGoogle, KindAnthropic, KindCompat:
		return true
	default:
		return false
	}
}

func keyHint(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 4 {
		return "…" + key
	}
	return "…" + key[len(key)-4:]
}

func splitKindModel(s string) (Kind, string, bool) {
	for _, k := range []Kind{KindCompat, KindAnthropic, KindOpenAI, KindGoogle, KindFake} {
		p := string(k) + ":"
		if strings.HasPrefix(s, p) {
			return k, s[len(p):], true
		}
	}
	return "", "", false
}

func settingsFromJSON(providers, models json.RawMessage) Settings {
	var out Settings
	var ps []storedProvider
	if len(providers) > 0 {
		_ = json.Unmarshal(providers, &ps)
	}
	if len(ps) > 0 {
		p := ps[0]
		out.Kind = p.Kind
		out.HasKey = p.SecretID != uuid.Nil
		out.Hint = p.Hint
		out.BaseURL = p.BaseURL
	}
	var mm map[string]string
	if len(models) > 0 {
		_ = json.Unmarshal(models, &mm)
	}
	if s := mm["compose"]; s != "" {
		if _, model, ok := splitKindModel(s); ok {
			out.ComposeModel = model
		}
	}
	if s := mm["embed"]; s != "" {
		if _, model, ok := splitKindModel(s); ok {
			out.EmbedModel = model
		}
	}
	return out
}

func validatePut(in PutInput, existing Settings) error {
	if !kindOK(in.Kind) {
		return ErrUnknownKind
	}
	if strings.TrimSpace(in.ComposeModel) == "" {
		return ErrComposeModel
	}
	if strings.TrimSpace(in.EmbedModel) == "" {
		return ErrEmbedModel
	}
	if in.Kind == KindAnthropic {
		return ErrAnthropicEmbed
	}
	if in.Kind == KindCompat && strings.TrimSpace(in.BaseURL) == "" {
		return ErrBaseURL
	}
	if strings.TrimSpace(in.APIKey) == "" && !existing.HasKey {
		return ErrKeyRequired
	}
	return nil
}
