package ai

import (
	"errors"
	"testing"
)

func TestResolveRejectsAnthropicEmbed(t *testing.T) {
	t.Parallel()
	_, _, err := Resolve(RoleEmbed, []Binding{{
		Role: RoleEmbed, Kind: KindAnthropic, Model: "claude-3", APIKey: "sk-ws",
	}}, nil)
	if !errors.Is(err, ErrAnthropicEmbed) {
		t.Fatalf("%v", err)
	}
}

func TestResolveBYOKKeyWinsOverPlatform(t *testing.T) {
	t.Parallel()
	b := FillKey(Binding{Kind: KindOpenAI, APIKey: "sk-workspace"}, Platform{OpenAIAPIKey: "sk-platform"})
	if b.APIKey != "sk-workspace" {
		t.Fatalf("got %q", b.APIKey)
	}
	b = FillKey(Binding{Kind: KindOpenAI}, Platform{OpenAIAPIKey: "sk-platform"})
	if b.APIKey != "sk-platform" {
		t.Fatalf("got %q", b.APIKey)
	}
}

func TestResolveFakeForClassify(t *testing.T) {
	t.Parallel()
	p, b, err := Resolve(RoleClassify, []Binding{{
		Role: RoleClassify, Kind: KindFake, Model: "fake-cls",
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := p.(*Fake); !ok || b.Model != "fake-cls" {
		t.Fatalf("%T %+v", p, b)
	}
}

func TestResolveFallback(t *testing.T) {
	t.Parallel()
	fb := Binding{Kind: KindFake, Model: "fb"}
	p, _, err := Resolve(RoleReply, nil, &fb)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := p.(*Fake); !ok {
		t.Fatalf("%T", p)
	}
}

func TestNewOpenAINeedsKey(t *testing.T) {
	t.Parallel()
	_, err := New(Binding{Kind: KindOpenAI})
	if !errors.Is(err, ErrNoKey) {
		t.Fatalf("%v", err)
	}
	p, err := New(Binding{Kind: KindOpenAI, APIKey: "sk", Model: "gpt-4o-mini"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := p.(*OpenAI); !ok {
		t.Fatalf("%T", p)
	}
	g, err := New(Binding{Kind: KindGoogle, APIKey: "studio"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := g.(*Gemini); !ok {
		t.Fatalf("%T", g)
	}
}
