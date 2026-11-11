package storage

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestDiskPutGetAcrossRoots(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ctx := context.Background()
	a, err := NewDisk(filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Put(ctx, "ws/r1", "application/pdf", []byte("%PDF")); err != nil {
		t.Fatal(err)
	}
	b, err := NewDisk(filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := b.Get(ctx, "ws/r1")
	if err != nil || !bytes.Equal(got, []byte("%PDF")) {
		t.Fatalf("%q %v", got, err)
	}
}

func TestDiskMissingAndBadKey(t *testing.T) {
	t.Parallel()
	s, err := NewDisk(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := s.Get(ctx, "nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("%v", err)
	}
	if err := s.Put(ctx, "../etc/passwd", "text/plain", []byte("x")); !errors.Is(err, ErrBadKey) {
		t.Fatalf("%v", err)
	}
	if err := s.Put(ctx, "", "text/plain", []byte("x")); !errors.Is(err, ErrBadKey) {
		t.Fatalf("%v", err)
	}
}

func TestOpenDiskAndMemory(t *testing.T) {
	t.Parallel()
	d, err := Open("disk", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := d.(*Disk); !ok {
		t.Fatalf("%T", d)
	}
	m, err := Open("memory", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := m.(*Memory); !ok {
		t.Fatalf("%T", m)
	}
	if _, err := Open("s3", ""); err == nil {
		t.Fatal("want unknown driver")
	}
}
