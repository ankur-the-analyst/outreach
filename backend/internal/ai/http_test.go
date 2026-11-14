package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"outreach/internal/auth"
)

type fakeStore struct {
	role string
	got  Settings
	err  error
	in   PutInput
}

func (f *fakeStore) Membership(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return f.role, f.err
}
func (f *fakeStore) Get(context.Context, uuid.UUID) (Settings, error) {
	return f.got, f.err
}
func (f *fakeStore) Put(_ context.Context, _ uuid.UUID, in PutInput) (Settings, error) {
	f.in = in
	if err := validatePut(in, f.got); err != nil {
		return Settings{}, err
	}
	hint := f.got.Hint
	if in.APIKey != "" {
		hint = keyHint(in.APIKey)
	}
	f.got = Settings{
		Kind: in.Kind, HasKey: true, Hint: hint, BaseURL: in.BaseURL,
		ComposeModel: in.ComposeModel, EmbedModel: in.EmbedModel,
	}
	return f.got, nil
}

func session(req *http.Request) *http.Request {
	return req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
}

func TestGetSettingsHTTPEmpty(t *testing.T) {
	t.Parallel()
	api := NewAPI(&fakeStore{role: "owner"})
	req := session(httptest.NewRequest(http.MethodGet, "/api/v1/settings/ai", nil))
	rec := httptest.NewRecorder()
	api.Get(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "api_key") {
		t.Fatalf("leaked key field: %s", rec.Body.String())
	}
	var view SettingsView
	if err := json.Unmarshal(rec.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if len(view.Providers) == 0 || view.Providers[0].Kind != KindOpenAI {
		t.Fatalf("%+v", view.Providers)
	}
}

func TestPutSettingsHTTP(t *testing.T) {
	t.Parallel()
	store := &fakeStore{role: "owner"}
	api := NewAPI(store)
	body := `{"kind":"openai","api_key":"sk-live-secret","compose_model":"gpt-6-luna","embed_model":"text-embedding-3-small"}`
	req := session(httptest.NewRequest(http.MethodPut, "/api/v1/settings/ai", bytes.NewBufferString(body)))
	rec := httptest.NewRecorder()
	api.Put(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	if store.in.APIKey != "sk-live-secret" {
		t.Fatalf("put %+v", store.in)
	}
	var out SettingsView
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if !out.HasKey || out.Hint != "…cret" || out.Kind != KindOpenAI {
		t.Fatalf("%+v", out)
	}
	if len(out.Providers) == 0 {
		t.Fatal("missing providers")
	}
	if strings.Contains(rec.Body.String(), "sk-live-secret") {
		t.Fatalf("leaked key: %s", rec.Body.String())
	}
}

func TestPutSettingsHTTPRejectsAnthropic(t *testing.T) {
	t.Parallel()
	api := NewAPI(&fakeStore{role: "owner"})
	body := `{"kind":"anthropic","api_key":"sk-ant","compose_model":"claude","embed_model":"claude"}`
	req := session(httptest.NewRequest(http.MethodPut, "/api/v1/settings/ai", bytes.NewBufferString(body)))
	rec := httptest.NewRecorder()
	api.Put(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
}

func TestPutSettingsHTTPKeepKey(t *testing.T) {
	t.Parallel()
	store := &fakeStore{role: "owner", got: Settings{HasKey: true, Hint: "…oldx", Kind: KindOpenAI}}
	api := NewAPI(store)
	body := `{"kind":"openai","compose_model":"gpt-6-luna","embed_model":"text-embedding-3-small"}`
	req := session(httptest.NewRequest(http.MethodPut, "/api/v1/settings/ai", bytes.NewBufferString(body)))
	rec := httptest.NewRecorder()
	api.Put(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	if store.in.APIKey != "" {
		t.Fatalf("unexpected key %q", store.in.APIKey)
	}
}
