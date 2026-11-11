package cv

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"outreach/internal/ai"
)

type stubBind struct {
	p   ai.Provider
	err error
}

func (s stubBind) Provider(context.Context, uuid.UUID, ai.Role) (ai.Provider, error) {
	return s.p, s.err
}

func TestJSONForUsesBinder(t *testing.T) {
	t.Parallel()
	got := &ai.Fake{JSON: []byte(`{"name":"Ada"}`)}
	p := NewPostgres(nil, nil).WithBinder(stubBind{p: got})
	w, err := p.jsonFor(context.Background(), uuid.Nil)
	if err != nil || w != got {
		t.Fatalf("%T %v", w, err)
	}
}

func TestJSONForSkipsWhenNoKey(t *testing.T) {
	t.Parallel()
	p := NewPostgres(nil, nil).
		WithJSON(&ai.Fake{}).
		WithBinder(stubBind{err: ai.ErrNoKey})
	w, err := p.jsonFor(context.Background(), uuid.Nil)
	if err != nil || w != nil {
		t.Fatalf("%T %v", w, err)
	}
}

func TestJSONForWithoutBinderKeepsJSON(t *testing.T) {
	t.Parallel()
	fb := &ai.Fake{}
	p := NewPostgres(nil, nil).WithJSON(fb)
	w, err := p.jsonFor(context.Background(), uuid.Nil)
	if err != nil || w != fb {
		t.Fatalf("%T %v", w, err)
	}
}
