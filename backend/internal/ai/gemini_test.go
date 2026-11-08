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

func TestNewGoogleNeedsKey(t *testing.T) {
	t.Parallel()
	_, err := New(Binding{Kind: KindGoogle})
	if !errors.Is(err, ErrNoKey) {
		t.Fatalf("%v", err)
	}
}

func TestGeminiComplete(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-goog-api-key") != "studio-key" && !strings.Contains(r.URL.RawQuery, "key=studio-key") {
			t.Fatalf("missing key header=%q query=%q", r.Header.Get("x-goog-api-key"), r.URL.RawQuery)
		}
		if !strings.Contains(r.URL.Path, "gemini-2.0-flash") || !strings.Contains(r.URL.Path, "generateContent") {
			t.Fatalf("%s", r.URL.Path)
		}
		raw, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(raw), "hello") {
			t.Fatalf("%s", raw)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{
				{"content": map[string]any{
					"parts": []map[string]string{{"text": "  From Gemini.  "}},
				}},
			},
		})
	}))
	t.Cleanup(srv.Close)
	p, err := New(Binding{
		Kind: KindGoogle, APIKey: "studio-key", Model: "gemini-2.0-flash",
		BaseURL: srv.URL, HTTP: srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := p.Complete(context.Background(), Request{Role: RoleCompose, System: "sys", User: "hello", Temperature: 0.6})
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "From Gemini." || got.Model != "gemini-2.0-flash" {
		t.Fatalf("%+v", got)
	}
}

func TestGeminiHTTPError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"bad key"}}`))
	}))
	t.Cleanup(srv.Close)
	p, err := New(Binding{Kind: KindGoogle, APIKey: "bad", BaseURL: srv.URL, HTTP: srv.Client()})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Complete(context.Background(), Request{User: "hi"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGeminiEmbedNotSupported(t *testing.T) {
	t.Parallel()
	p, err := New(Binding{Kind: KindGoogle, APIKey: "k"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Embed(context.Background(), []string{"a"})
	if !errors.Is(err, ErrNotSupported) {
		t.Fatalf("%v", err)
	}
}
