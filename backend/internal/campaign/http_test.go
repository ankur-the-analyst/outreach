package campaign

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
	c    Campaign
	err  error
}

func (f *fakeStore) Membership(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return f.role, nil
}
func (f *fakeStore) Packs() []Pack { return nil }
func (f *fakeStore) PutIdentity(context.Context, uuid.UUID, Identity) (Identity, error) {
	return Identity{}, nil
}
func (f *fakeStore) GetIdentity(context.Context, uuid.UUID) (Identity, error) {
	return Identity{}, ErrNotFound
}
func (f *fakeStore) Create(context.Context, uuid.UUID, CreateInput) (Campaign, error) {
	return f.c, nil
}
func (f *fakeStore) List(context.Context, uuid.UUID) ([]Campaign, error) { return nil, nil }
func (f *fakeStore) Get(context.Context, uuid.UUID, uuid.UUID) (Campaign, error) {
	return f.c, nil
}
func (f *fakeStore) Start(context.Context, uuid.UUID, uuid.UUID) (Campaign, error) {
	if f.err != nil {
		return Campaign{}, f.err
	}
	return f.c, nil
}
func (f *fakeStore) Pause(context.Context, uuid.UUID, uuid.UUID) (Campaign, error) {
	return f.c, nil
}
func (f *fakeStore) Stop(context.Context, uuid.UUID, uuid.UUID) (Campaign, error) {
	return f.c, nil
}
func (f *fakeStore) Enrol(context.Context, uuid.UUID, uuid.UUID, EnrolInput) (Enrolment, error) {
	return Enrolment{}, nil
}
func (f *fakeStore) ListEnrolments(context.Context, uuid.UUID, uuid.UUID) ([]Enrolment, error) {
	return nil, nil
}
func (f *fakeStore) ListTracker(context.Context, uuid.UUID) ([]TrackerRow, error) {
	return nil, nil
}

func TestStartCommercialWithoutIdentity(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("01700000-0000-7000-8000-000000000003")
	api := New(&fakeStore{role: "owner", err: ErrIdentity})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns/"+id.String()+"/start", nil)
	req.SetPathValue("id", id.String())
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := httptest.NewRecorder()
	api.Start(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("sender identity required")) {
		t.Fatalf("%s", rec.Body.String())
	}
}
