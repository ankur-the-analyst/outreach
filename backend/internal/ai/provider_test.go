package ai

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestFakeProvider(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	f := &Fake{
		Model: "fake-1",
		Text:  "hello",
		JSON:  json.RawMessage(`{"class":"other"}`),
		Vectors: [][]float32{
			{0.1, 0.2},
		},
	}
	got, err := f.Complete(ctx, Request{Role: RoleCompose, User: "hi"})
	if err != nil || got.Text != "hello" || got.Model != "fake-1" {
		t.Fatalf("%+v %v", got, err)
	}
	js, err := f.CompleteJSON(ctx, Request{Role: RoleClassify}, json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Class string `json:"class"`
	}
	if err := json.Unmarshal(js.Raw, &payload); err != nil || payload.Class != "other" {
		t.Fatalf("%s %v", js.Raw, err)
	}
	vecs, err := f.Embed(ctx, []string{"a"})
	if err != nil || len(vecs) != 1 || vecs[0][0] != 0.1 {
		t.Fatalf("%v %v", vecs, err)
	}
}

func TestUseSiteTakesJSONOnly(t *testing.T) {
	t.Parallel()
	type jsonCompleter interface {
		CompleteJSON(context.Context, Request, json.RawMessage) (JSONResult, error)
	}
	var p jsonCompleter = &Fake{JSON: json.RawMessage(`{"ok":true}`)}
	got, err := p.CompleteJSON(context.Background(), Request{Role: RoleClassify}, nil)
	if err != nil || string(got.Raw) != `{"ok":true}` {
		t.Fatalf("%s %v", got.Raw, err)
	}
}

func TestAnthropicCannotEmbed(t *testing.T) {
	t.Parallel()
	_, err := Anthropic{}.Embed(context.Background(), []string{"x"})
	if !errors.Is(err, ErrAnthropicEmbed) {
		t.Fatalf("%v", err)
	}
}

func TestTemperature(t *testing.T) {
	t.Parallel()
	if Temperature(RoleClassify) != 0.2 || Temperature(RoleCompose) != 0.6 {
		t.Fatal("unexpected temperatures")
	}
}
