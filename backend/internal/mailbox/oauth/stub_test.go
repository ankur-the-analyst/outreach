package oauth

import (
	"context"
	"errors"
	"testing"

	"outreach/internal/mailbox"
)

func TestStubNotConfigured(t *testing.T) {
	t.Parallel()
	var s Stub
	err := s.Test(context.Background())
	if !errors.Is(err, mailbox.ErrNotConfigured) {
		t.Fatalf("got %v", err)
	}
	_, err = s.Send(context.Background(), nil)
	if !errors.Is(err, mailbox.ErrNotConfigured) {
		t.Fatalf("send %v", err)
	}
}
