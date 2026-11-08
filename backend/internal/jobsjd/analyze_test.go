package jobsjd

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"outreach/internal/ai"
)

func TestJSONWriterFallsBackWithoutBinding(t *testing.T) {
	t.Parallel()
	fb := &ai.Fake{JSON: json.RawMessage(`{"summary":"ok"}`)}
	p := NewPostgres(nil, nil, fb).WithBinder(stubBind{err: ai.ErrNoBinding})
	w, err := p.jsonWriter(context.Background(), uuid.Nil)
	if err != nil || w != fb {
		t.Fatalf("%T %v", w, err)
	}
}

type stubBind struct {
	p   ai.Provider
	err error
}

func (s stubBind) Provider(context.Context, uuid.UUID, ai.Role) (ai.Provider, error) {
	return s.p, s.err
}
