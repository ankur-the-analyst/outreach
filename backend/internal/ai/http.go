package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"outreach/internal/auth"
)

type Store interface {
	Membership(ctx context.Context, workspaceID, userID uuid.UUID) (string, error)
	Get(ctx context.Context, workspaceID uuid.UUID) (Settings, error)
	Put(ctx context.Context, workspaceID uuid.UUID, in PutInput) (Settings, error)
}

type API struct {
	store Store
}

func NewAPI(store Store) *API {
	return &API{store: store}
}

func (a *API) Get(w http.ResponseWriter, r *http.Request) {
	s, ok := a.member(w, r)
	if !ok {
		return
	}
	out, err := a.store.Get(r.Context(), s.WorkspaceID)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) Put(w http.ResponseWriter, r *http.Request) {
	s, ok := a.edit(w, r)
	if !ok {
		return
	}
	var in struct {
		Kind         string `json:"kind"`
		APIKey       string `json:"api_key"`
		BaseURL      string `json:"base_url"`
		ComposeModel string `json:"compose_model"`
		EmbedModel   string `json:"embed_model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json")
		return
	}
	out, err := a.store.Put(r.Context(), s.WorkspaceID, PutInput{
		Kind:         Kind(in.Kind),
		APIKey:       in.APIKey,
		BaseURL:      in.BaseURL,
		ComposeModel: in.ComposeModel,
		EmbedModel:   in.EmbedModel,
	})
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) member(w http.ResponseWriter, r *http.Request) (auth.Session, bool) {
	s, ok := auth.SessionFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthenticated", "sign in required")
		return auth.Session{}, false
	}
	if _, err := a.store.Membership(r.Context(), s.WorkspaceID, s.UserID); err != nil {
		writeStoreError(w, err)
		return auth.Session{}, false
	}
	return s, true
}

func (a *API) edit(w http.ResponseWriter, r *http.Request) (auth.Session, bool) {
	s, ok := a.member(w, r)
	if !ok {
		return auth.Session{}, false
	}
	role, err := a.store.Membership(r.Context(), s.WorkspaceID, s.UserID)
	if err != nil {
		writeStoreError(w, err)
		return auth.Session{}, false
	}
	if role != "owner" && role != "member" {
		writeStoreError(w, ErrForbidden)
		return auth.Session{}, false
	}
	return s, true
}

func writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "owner or member role required")
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "not found")
	case errors.Is(err, ErrUnknownKind):
		writeError(w, http.StatusBadRequest, "invalid_input", "unknown model provider")
	case errors.Is(err, ErrAnthropicEmbed):
		writeError(w, http.StatusBadRequest, "invalid_input", "Embed cannot use Anthropic")
	case errors.Is(err, ErrKeyRequired):
		writeError(w, http.StatusBadRequest, "invalid_input", "Paste an API key")
	case errors.Is(err, ErrBaseURL):
		writeError(w, http.StatusBadRequest, "invalid_input", "Base URL is required for an OpenAI-compatible provider")
	case errors.Is(err, ErrComposeModel), errors.Is(err, ErrEmbedModel):
		writeError(w, http.StatusBadRequest, "invalid_input", "Compose and embed models are required")
	default:
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{"code": code, "message": message},
	})
}
