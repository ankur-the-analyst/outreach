package mailbox

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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
func (f *fakeStore) ListDomains(context.Context, uuid.UUID) ([]SendingDomain, error) {
	return []SendingDomain{{Domain: "acme.com", DailyLimit: 40}}, f.err
}
func (f *fakeStore) ListWarmups(context.Context, uuid.UUID) ([]Warmup, error) {
	n := int32(2)
	return []Warmup{{Name: "Standard", Kind: "standard", FirstDay: &n}}, f.err
}
func (f *fakeStore) CreateWarmup(_ context.Context, _ uuid.UUID, in CreateWarmupInput) (Warmup, error) {
	if strings.TrimSpace(in.Name) == "" {
		return Warmup{}, ErrWarmupName
	}
	if f.err != nil {
		return Warmup{}, f.err
	}
	n := int32(in.Start)
	return Warmup{Name: in.Name, Kind: "custom", FirstDay: &n}, nil
}
func (f *fakeStore) Connect(_ context.Context, _ uuid.UUID, in ConnectInput) (Mailbox, error) {
	f.connect = in
	if f.err != nil {
		return Mailbox{}, f.err
	}
	return Mailbox{Email: in.Email, AuthKind: in.AuthKind}, nil
}
func (f *fakeStore) Pause(context.Context, uuid.UUID, uuid.UUID) error     { return f.err }
func (f *fakeStore) Reconnect(context.Context, uuid.UUID, uuid.UUID) error { return f.err }
func (f *fakeStore) AssignWarmup(context.Context, uuid.UUID, uuid.UUID, string, int) error {
	return f.err
}
func (f *fakeStore) SetReplyMode(context.Context, uuid.UUID, uuid.UUID, string) error {
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
func (f *fakeStore) ListPools(context.Context, uuid.UUID) ([]Pool, error) { return nil, f.err }
func (f *fakeStore) CreatePool(_ context.Context, _ uuid.UUID, name string, ids []uuid.UUID) (Pool, error) {
	if f.err != nil {
		return Pool{}, f.err
	}
	return Pool{Name: name, Mailboxes: make([]PoolMailbox, len(ids))}, nil
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

func TestPatchRejectsAutoReplyMode(t *testing.T) {
	t.Parallel()
	api := New(&fakeStore{role: "owner", err: ErrReplyMode}, fakeTester{})
	id := uuid.MustParse("01700000-0000-7000-8000-000000000003")
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/mailboxes/"+id.String(), bytes.NewBufferString(`{"reply_mode":"auto"}`))
	req.SetPathValue("id", id.String())
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.Patch(rec, req)
	if rec.Code != http.StatusBadRequest || !bytes.Contains(rec.Body.Bytes(), []byte("reply_mode")) {
		t.Fatalf("got %d %s", rec.Code, rec.Body.Bytes())
	}
}

func TestPatchWarmupKind(t *testing.T) {
	t.Parallel()
	api := New(&fakeStore{role: "owner"}, fakeTester{})
	id := uuid.MustParse("01700000-0000-7000-8000-000000000003")
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/mailboxes/"+id.String(), bytes.NewBufferString(`{"warmup_kind":"conservative"}`))
	req.SetPathValue("id", id.String())
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.Patch(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("got %d %s", rec.Code, rec.Body.Bytes())
	}
}

func TestListDomainsHTTP(t *testing.T) {
	t.Parallel()
	api := New(&fakeStore{role: "owner"}, fakeTester{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sending-domains", nil)
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.ListDomains(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d %s", rec.Code, rec.Body.Bytes())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"acme.com"`)) {
		t.Fatalf("body %s", rec.Body.Bytes())
	}
}

func TestListWarmupsHTTP(t *testing.T) {
	t.Parallel()
	api := New(&fakeStore{role: "owner"}, fakeTester{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/warmup-profiles", nil)
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.ListWarmups(rec, req)
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte(`"first_day":2`)) {
		t.Fatalf("got %d %s", rec.Code, rec.Body.Bytes())
	}
}

func TestCreateWarmupHTTP(t *testing.T) {
	t.Parallel()
	api := New(&fakeStore{role: "owner"}, fakeTester{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/warmup-profiles", bytes.NewBufferString(`{"name":"Slow","start":3}`))
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.CreateWarmup(rec, req)
	if rec.Code != http.StatusCreated || !bytes.Contains(rec.Body.Bytes(), []byte(`"custom"`)) {
		t.Fatalf("got %d %s", rec.Code, rec.Body.Bytes())
	}
}

func TestCreateWarmupRequiresName(t *testing.T) {
	t.Parallel()
	api := New(&fakeStore{role: "owner"}, fakeTester{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/warmup-profiles", bytes.NewBufferString(`{"name":" ","start":2}`))
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.CreateWarmup(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d %s", rec.Code, rec.Body.Bytes())
	}
}

func TestCreatePoolRequiresName(t *testing.T) {
	t.Parallel()
	api := New(&fakeStore{role: "owner"}, fakeTester{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/mailbox-pools", bytes.NewBufferString(`{"name":"  "}`))
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.CreatePool(rec, req)
	if rec.Code != http.StatusBadRequest {
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
