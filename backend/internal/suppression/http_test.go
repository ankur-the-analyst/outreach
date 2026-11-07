package suppression

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"outreach/internal/auth"
)

type fakeStore struct {
	role string
	ent  Entry
	err  error
}

func (f *fakeStore) Membership(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return f.role, f.err
}
func (f *fakeStore) Add(context.Context, uuid.UUID, string, string, string) (Entry, error) {
	return f.ent, f.err
}

func TestCreateSuppression(t *testing.T) {
	t.Parallel()
	api := New(&fakeStore{role: "owner", ent: Entry{Kind: KindEmail, Value: "a@b.co", Reason: "opt-out"}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/suppression", bytes.NewBufferString(`{"kind":"email","value":"a@b.co","reason":"opt-out"}`))
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000001"), UserID: uuid.MustParse("01700000-0000-7000-8000-000000000002")}))
	rec := httptest.NewRecorder()
	api.Create(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
}
