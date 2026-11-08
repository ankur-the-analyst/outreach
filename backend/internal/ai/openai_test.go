package ai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAIComplete(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("%s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test" {
			t.Fatalf("auth %s", got)
		}
		raw, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(raw), `"gpt-4o-mini"`) {
			t.Fatalf("%s", raw)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model": "gpt-4o-mini",
			"choices": []map[string]any{
				{"message": map[string]string{"content": "  Hello from BYOK.  "}},
			},
		})
	}))
	t.Cleanup(srv.Close)
	p := &OpenAI{Model: "gpt-4o-mini", APIKey: "sk-test", BaseURL: srv.URL, HTTP: srv.Client()}
	got, err := p.Complete(context.Background(), Request{Role: RoleCompose, System: "sys", User: "hi", Temperature: 0.6})
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "Hello from BYOK." || got.Model != "gpt-4o-mini" {
		t.Fatalf("%+v", got)
	}
}

func TestOpenAIHTTPError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)
	p := &OpenAI{APIKey: "sk", BaseURL: srv.URL, HTTP: srv.Client()}
	_, err := p.Complete(context.Background(), Request{User: "hi"})
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("%v", err)
	}
}

func TestOpenAIEmbedNotSupported(t *testing.T) {
	t.Parallel()
	_, err := (&OpenAI{}).Embed(context.Background(), []string{"a"})
	if !errors.Is(err, ErrNotSupported) {
		t.Fatalf("%v", err)
	}
}
