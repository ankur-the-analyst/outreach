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

func TestWriterErrorsWithoutBinding(t *testing.T) {
	t.Parallel()
	fb := &ai.Fake{Text: "fallback"}
	p := NewPostgres(nil, nil).WithWriter(fb).WithBinder(stubBind{err: ai.ErrNoBinding})
	_, err := p.writer(context.Background(), uuid.Nil)
	if !errors.Is(err, ai.ErrNoBinding) {
		t.Fatalf("%v", err)
	}
}

func TestWriterErrorsWithoutKey(t *testing.T) {
	t.Parallel()
	p := NewPostgres(nil, nil).WithWriter(&ai.Fake{Text: "fallback"}).WithBinder(stubBind{err: ai.ErrNoKey})
	_, err := p.writer(context.Background(), uuid.Nil)
	if !errors.Is(err, ai.ErrNoKey) {
		t.Fatalf("%v", err)
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

func TestWriterAllowsTestFakeWithoutBinder(t *testing.T) {
	t.Parallel()
	fb := &ai.Fake{Text: "test"}
	p := NewPostgres(nil, nil).WithWriter(fb)
	w, err := p.writer(context.Background(), uuid.Nil)
	if err != nil || w != fb {
		t.Fatalf("%v %T", err, w)
	}
}
