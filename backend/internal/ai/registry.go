package ai

import "strings"

// ProviderSpec is one entry in the workspace model catalog.
// Fake is not listed; tests construct it in process.
type ProviderSpec struct {
	Kind         Kind     `json:"kind"`
	Label        string   `json:"label"`
	NeedsBaseURL bool     `json:"needs_base_url"`
	Embed        bool     `json:"embed"`
	CustomModels bool     `json:"custom_models,omitempty"`
	Compose      []string `json:"compose_models"`
	EmbedModels  []string `json:"embed_models"`
}

var registry = []ProviderSpec{
	{
		Kind:  KindOpenAI,
		Label: "OpenAI",
		Embed: true,
		Compose: []string{
			"gpt-6.1-sol",
			"gpt-6.1-terra",
			"gpt-6-luna",
			"gpt-5.6-sol",
		},
		EmbedModels: []string{
			"text-embedding-3-small",
			"text-embedding-3-large",
		},
	},
	{
		Kind:  KindGoogle,
		Label: "Google",
		Embed: true,
		Compose: []string{
			"gemini-3.8-flash",
			"gemini-3.5-flash-lite",
			"gemini-2.5-flash",
			"gemini-2.5-pro",
		},
		EmbedModels: []string{
			"gemini-embedding-001",
		},
	},
	{
		Kind:        KindAnthropic,
		Label:       "Anthropic",
		Embed:       false,
		Compose:     []string{"claude-sonnet-5-5", "claude-haiku-5-5", "claude-opus-5-5"},
		EmbedModels: []string{},
	},
	{
		Kind:         KindCompat,
		Label:        "OpenAI-compatible",
		NeedsBaseURL: true,
		Embed:        true,
		CustomModels: true,
	},
}

func Registry() []ProviderSpec {
	out := make([]ProviderSpec, len(registry))
	copy(out, registry)
	return out
}

func Lookup(k Kind) (ProviderSpec, bool) {
	for _, p := range registry {
		if p.Kind == k {
			return p, true
		}
	}
	return ProviderSpec{}, false
}

func modelListed(list []string, model string) bool {
	for _, m := range list {
		if m == model {
			return true
		}
	}
	return false
}

func checkModel(spec ProviderSpec, role Role, model string) error {
	model = strings.TrimSpace(model)
	if model == "" {
		if role == RoleEmbed {
			return ErrEmbedModel
		}
		return ErrComposeModel
	}
	if spec.CustomModels {
		return nil
	}
	switch role {
	case RoleEmbed:
		if !spec.Embed || !modelListed(spec.EmbedModels, model) {
			return ErrUnknownModel
		}
	default:
		if !modelListed(spec.Compose, model) {
			return ErrUnknownModel
		}
	}
	return nil
}
