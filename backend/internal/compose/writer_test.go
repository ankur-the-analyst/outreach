package compose

import (
	"context"
	"errors"
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

func TestWriterFallsBackWithoutBinding(t *testing.T) {
	t.Parallel()
	fb := &ai.Fake{Text: "fallback"}
	p := NewPostgres(nil, nil).WithWriter(fb).WithBinder(stubBind{err: ai.ErrNoBinding})
	w, err := p.writer(context.Background(), uuid.Nil)
	if err != nil {
		t.Fatal(err)
	}
	if w != fb {
		t.Fatalf("%T", w)
	}
}

func TestWriterUsesBinder(t *testing.T) {
	t.Parallel()
	byok := &ai.Fake{Text: "byok"}
	p := NewPostgres(nil, nil).WithWriter(&ai.Fake{Text: "fallback"}).WithBinder(stubBind{p: byok})
	w, err := p.writer(context.Background(), uuid.Nil)
	if err != nil {
		t.Fatal(err)
	}
	if w != byok {
		t.Fatalf("%T", w)
	}
}

func TestWriterSurfacesGoogle(t *testing.T) {
	t.Parallel()
	p := NewPostgres(nil, nil).WithWriter(&ai.Fake{Text: "fallback"}).WithBinder(stubBind{err: ai.ErrNotConfigured})
	_, err := p.writer(context.Background(), uuid.Nil)
	if !errors.Is(err, ai.ErrNotConfigured) {
		t.Fatalf("%v", err)
	}
}
