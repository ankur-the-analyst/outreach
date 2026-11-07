package sender

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"outreach/internal/auth"
	"outreach/internal/pacing"
)

type fakeStore struct {
	role string
	ent  Entry
	used int64
	err  error
}

func (f *fakeStore) Membership(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return f.role, f.err
}
func (f *fakeStore) Hold(context.Context, uuid.UUID, HoldInput) (Entry, error) {
	return f.ent, f.err
}
func (f *fakeStore) Cancel(context.Context, uuid.UUID, uuid.UUID) (Entry, error) {
	return f.ent, f.err
}
func (f *fakeStore) Usage(context.Context, uuid.UUID, uuid.UUID) (int64, error) {
	return f.used, f.err
}
func (f *fakeStore) Reserve(context.Context, uuid.UUID, ReserveInput) (Entry, error) {
	return f.ent, f.err
}
func (f *fakeStore) Send(context.Context, uuid.UUID, uuid.UUID, SendInput) (Entry, error) {
	return f.ent, f.err
}

func TestHoldCreated(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("01700000-0000-7000-8000-000000000003")
	api := New(&fakeStore{role: "owner", ent: Entry{ID: id, State: pacing.StateHeld}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mailboxes/"+id.String()+"/holds", bytes.NewBufferString(`{"rfc_message_id":"<m@x>"}`))
	req.SetPathValue("id", id.String())
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.Hold(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("got %d %s", rec.Code, rec.Body.Bytes())
	}
}

func TestCancelConflict(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("01700000-0000-7000-8000-000000000003")
	api := New(&fakeStore{role: "owner", err: ErrConflict})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ledger/"+id.String()+"/cancel", nil)
	req.SetPathValue("id", id.String())
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.Cancel(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("got %d", rec.Code)
	}
}
