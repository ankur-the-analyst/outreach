package events

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"outreach/internal/auth"
)

type fakeStore struct {
	n int64
}

func (f *fakeStore) Membership(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	return "owner", nil
}
func (f *fakeStore) ReviewN(context.Context, uuid.UUID) (int64, error) {
	return f.n, nil
}

type flushRec struct {
	*httptest.ResponseRecorder
}

func (flushRec) Flush() {}

func TestStreamWritesReviewThenStops(t *testing.T) {
	t.Parallel()
	api := New(&fakeStore{n: 4})
	api.every = time.Hour
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/stream", nil).WithContext(ctx)
	req = req.WithContext(auth.WithSession(req.Context(), auth.Session{
		UserID:      uuid.MustParse("01700000-0000-7000-8000-000000000001"),
		WorkspaceID: uuid.MustParse("01700000-0000-7000-8000-000000000002"),
	}))
	rec := flushRec{httptest.NewRecorder()}
	done := make(chan struct{})
	go func() {
		api.Stream(rec, req)
		close(done)
	}()
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done
	if rec.Header().Get("Content-Type") != "text/event-stream" {
		t.Fatalf("ctype %s", rec.Header().Get("Content-Type"))
	}
	if !strings.Contains(rec.Body.String(), `"n":4`) {
		t.Fatalf("body %s", rec.Body.String())
	}
}
