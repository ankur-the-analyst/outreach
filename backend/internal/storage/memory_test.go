package storage

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

func TestMemoryPutGet(t *testing.T) {
	t.Parallel()
	s := NewMemory()
	ctx := context.Background()
	if err := s.Put(ctx, "ws/r1", "text/plain", []byte("hello")); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(ctx, "ws/r1")
	if err != nil || !bytes.Equal(got, []byte("hello")) {
		t.Fatalf("%q %v", got, err)
	}
	if _, err := s.Get(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("%v", err)
	}
}
