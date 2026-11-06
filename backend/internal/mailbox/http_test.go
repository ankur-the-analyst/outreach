package mailbox

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
	role    string
	connect ConnectInput
	err     error
}

func (f *fakeStore) Membership(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return f.role, nil
}
func (f *fakeStore) List(context.Context, uuid.UUID) ([]Mailbox, error) { return nil, nil }
func (f *fakeStore) Connect(_ context.Context, _ uuid.UUID, in ConnectInput) (Mailbox, error) {
	f.connect = in
	if f.err != nil {
		return Mailbox{}, f.err
	}
	return Mailbox{Email: in.Email, AuthKind: in.AuthKind}, nil
}
func (f *fakeStore) Pause(context.Context, uuid.UUID, uuid.UUID) error     { return f.err }
func (f *fakeStore) Reconnect(context.Context, uuid.UUID, uuid.UUID) error { return f.err }
func (f *fakeStore) AssignWarmup(context.Context, uuid.UUID, uuid.UUID, string) error {
	return f.err
}
func (f *fakeStore) CreateSA(context.Context, uuid.UUID, SAInput) (ServiceAccount, error) {
	return ServiceAccount{}, f.err
}
func (f *fakeStore) ListSA(context.Context, uuid.UUID) ([]ServiceAccount, error) {
	return nil, f.err
}
func (f *fakeStore) SMTPEndpoint(context.Context, uuid.UUID, uuid.UUID) (string, int, error) {
	return "127.0.0.1", 1025, f.err
}

type fakeTester struct{ err error }

func (f fakeTester) Test(context.Context, string, int) error { return f.err }

func TestCreateRejectsGmailServiceAccount(t *testing.T) {
	t.Parallel()
	store := &fakeStore{role: "owner", err: ErrGmailServiceAccount}
	api := New(store, fakeTester{})
	body := `{"email":"me@gmail.com","auth_kind":"service_account","timezone":"UTC"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mailboxes", bytes.NewBufferString(body))
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.Create(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d %s", rec.Code, rec.Body.Bytes())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("gmail_sa")) {
		t.Fatalf("body %s", rec.Body.Bytes())
	}
}

func TestCreateOAuthNotConfigured(t *testing.T) {
	t.Parallel()
	api := New(&fakeStore{role: "owner"}, fakeTester{})
	body := `{"email":"ada@acme.com","auth_kind":"oauth","timezone":"UTC"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mailboxes", bytes.NewBufferString(body))
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.Create(rec, req)
	if rec.Code != http.StatusBadRequest || !bytes.Contains(rec.Body.Bytes(), []byte("not_configured")) {
		t.Fatalf("got %d %s", rec.Code, rec.Body.Bytes())
	}
}

func TestCreateEmailTakenIsConflict(t *testing.T) {
	t.Parallel()
	api := New(&fakeStore{role: "owner", err: ErrEmailTaken}, fakeTester{})
	body := `{"email":"ada@acme.com","auth_kind":"smtp_imap","timezone":"UTC"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mailboxes", bytes.NewBufferString(body))
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.Create(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("got %d %s", rec.Code, rec.Body.Bytes())
	}
}

func TestMailboxSMTPTestNoSMTP(t *testing.T) {
	t.Parallel()
	api := New(&fakeStore{role: "owner", err: ErrNoSMTP}, fakeTester{})
	id := uuid.MustParse("01700000-0000-7000-8000-000000000003")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mailboxes/"+id.String()+"/test", nil)
	req.SetPathValue("id", id.String())
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.Test(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d %s", rec.Code, rec.Body.Bytes())
	}
}

func TestMailboxSMTPTestOK(t *testing.T) {
	t.Parallel()
	api := New(&fakeStore{role: "owner"}, fakeTester{})
	id := uuid.MustParse("01700000-0000-7000-8000-000000000003")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mailboxes/"+id.String()+"/test", nil)
	req.SetPathValue("id", id.String())
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.Test(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d %s", rec.Code, rec.Body.Bytes())
	}
}

func TestCreateRequiresTimezone(t *testing.T) {
	t.Parallel()
	api := New(&fakeStore{role: "owner"}, fakeTester{})
	body := `{"email":"me@x.co","auth_kind":"app_password","password":"x"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mailboxes", bytes.NewBufferString(body))
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.Create(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d", rec.Code)
	}
}
