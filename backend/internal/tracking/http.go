package tracking

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
)

// 1×1 GIF so a privacy proxy still gets a valid image.
var pixelGIF = []byte{
	0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x01, 0x00, 0x01, 0x00, 0x80, 0x00, 0x00,
	0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x21, 0xf9, 0x04, 0x01, 0x00, 0x00, 0x00,
	0x00, 0x2c, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x02, 0x02,
	0x44, 0x01, 0x00, 0x3b,
}

type Store interface {
	RecordFromDraft(ctx context.Context, workspaceID, draftID uuid.UUID, kind, url string) error
}

type API struct {
	store Store
	hmac  Hasher
}

func New(store Store, hmac Hasher) *API {
	return &API{store: store, hmac: hmac}
}

func (a *API) Open(w http.ResponseWriter, r *http.Request) {
	tok, err := ParseToken(a.hmac, r.PathValue("token"))
	if err != nil || tok.Kind != KindOpen {
		writePixel(w, http.StatusNoContent)
		return
	}
	_ = a.store.RecordFromDraft(r.Context(), tok.WorkspaceID, tok.DraftID, KindOpen, "")
	writePixel(w, http.StatusOK)
}

func (a *API) Click(w http.ResponseWriter, r *http.Request) {
	tok, err := ParseToken(a.hmac, r.PathValue("token"))
	if err != nil || tok.Kind != KindClick {
		http.NotFound(w, r)
		return
	}
	dest, ok := safeURL(tok.URL)
	if !ok {
		http.NotFound(w, r)
		return
	}
	_ = a.store.RecordFromDraft(r.Context(), tok.WorkspaceID, tok.DraftID, KindClick, tok.URL)
	w.Header().Set("Location", dest)
	w.WriteHeader(http.StatusFound)
}

func safeURL(raw string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return "", false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", false
	}
	return u.String(), true
}

func writePixel(w http.ResponseWriter, code int) {
	w.Header().Set("Content-Type", "image/gif")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	if code == http.StatusOK {
		_, _ = w.Write(pixelGIF)
	}
}
