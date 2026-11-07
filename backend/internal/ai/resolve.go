package ai

import (
	"fmt"
)

type Platform struct {
	OpenAIAPIKey string
	GeminiAPIKey string
}

func FillKey(b Binding, plat Platform) Binding {
	if b.APIKey != "" {
		return b
	}
	switch b.Kind {
	case KindOpenAI, KindCompat:
		b.APIKey = plat.OpenAIAPIKey
	case KindGoogle:
		b.APIKey = plat.GeminiAPIKey
	}
	return b
}

func Resolve(role Role, bindings []Binding, fallback *Binding) (Provider, Binding, error) {
	b, ok := bindingFor(role, bindings)
	if !ok && fallback != nil {
		b = *fallback
		b.Role = role
		ok = true
	}
	if !ok {
		return nil, Binding{}, fmt.Errorf("%w: %s", ErrNoBinding, role)
	}
	if role == RoleEmbed && b.Kind == KindAnthropic {
		return nil, Binding{}, fmt.Errorf("%w", ErrAnthropicEmbed)
	}
	p, err := New(b)
	if err != nil {
		return nil, Binding{}, err
	}
	return p, b, nil
}

func bindingFor(role Role, bindings []Binding) (Binding, bool) {
	for _, b := range bindings {
		if b.Role == role {
			return b, true
		}
	}
	return Binding{}, false
}
