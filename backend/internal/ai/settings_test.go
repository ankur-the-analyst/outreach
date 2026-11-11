package ai

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestValidatePutRejectsAnthropic(t *testing.T) {
	t.Parallel()
	err := validatePut(PutInput{
		Kind: KindAnthropic, APIKey: "x", ComposeModel: "claude", EmbedModel: "claude",
	}, Settings{})
	if !errors.Is(err, ErrAnthropicEmbed) {
		t.Fatalf("%v", err)
	}
}

func TestValidatePutRejectsUnknownModel(t *testing.T) {
	t.Parallel()
	err := validatePut(PutInput{
		Kind: KindOpenAI, APIKey: "sk", ComposeModel: "mystery", EmbedModel: "text-embedding-3-small",
	}, Settings{})
	if !errors.Is(err, ErrUnknownModel) {
		t.Fatalf("%v", err)
	}
}

func TestValidatePutRequiresKeyUntilSaved(t *testing.T) {
	t.Parallel()
	in := PutInput{Kind: KindOpenAI, ComposeModel: "gpt-4o-mini", EmbedModel: "text-embedding-3-small"}
	if err := validatePut(in, Settings{}); !errors.Is(err, ErrKeyRequired) {
		t.Fatalf("%v", err)
	}
	if err := validatePut(in, Settings{HasKey: true}); err != nil {
		t.Fatal(err)
	}
}

func TestSplitKindModelCompat(t *testing.T) {
	t.Parallel()
	k, m, ok := splitKindModel("openai_compat:local-embed")
	if !ok || k != KindCompat || m != "local-embed" {
		t.Fatalf("%s %s %v", k, m, ok)
	}
}

func TestSettingsFromJSONOmitsSecret(t *testing.T) {
	t.Parallel()
	sid := uuid.MustParse("01700000-0000-7000-8000-000000000009")
	providers, _ := json.Marshal([]storedProvider{{
		ID: sid, Kind: KindOpenAI, SecretID: sid, Hint: "…cret",
	}})
	models, _ := json.Marshal(map[string]string{
		"compose": "openai:gpt-4o-mini",
		"embed":   "openai:text-embedding-3-small",
	})
	got := settingsFromJSON(providers, models)
	if !got.HasKey || got.Hint != "…cret" || got.ComposeModel != "gpt-4o-mini" {
		t.Fatalf("%+v", got)
	}
}

func TestKeyHint(t *testing.T) {
	t.Parallel()
	if keyHint("sk-live-secret") != "…cret" {
		t.Fatalf("%q", keyHint("sk-live-secret"))
	}
}
