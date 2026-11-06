package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type stubPinger struct {
	err error
}

func (s stubPinger) Ping(context.Context) error { return s.err }

func TestHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		code int
		body string
	}{
		{name: "ok", code: http.StatusOK, body: "ok"},
		{name: "down", err: errors.New("connection refused"), code: http.StatusServiceUnavailable, body: "unhealthy"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			rec := httptest.NewRecorder()
			Handler(stubPinger{err: tt.err})(rec, req)

			if rec.Code != tt.code {
				t.Fatalf("status: got %d want %d", rec.Code, tt.code)
			}
			var got map[string]string
			if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if got["status"] != tt.body {
				t.Fatalf("body: got %q want %q", got["status"], tt.body)
			}
		})
	}
}
