package gmailapi

import (
	"context"
	"errors"
	"testing"

	"outreach/internal/mailbox"
)

func TestParseAndTest(t *testing.T) {
	t.Parallel()
	key, err := ParseKey([]byte(`{"client_email":"sa@proj.iam.gserviceaccount.com","client_id":"1","private_key":"-----BEGIN PRIVATE KEY-----\nM\n-----END PRIVATE KEY-----\n"}`))
	if err != nil {
		t.Fatal(err)
	}
	c := New("acme.com", "ada@acme.com", key)
	if err := c.Test(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestRejectsGmailDomain(t *testing.T) {
	t.Parallel()
	c := New("gmail.com", "ada@gmail.com", Key{ClientEmail: "sa@x", PrivateKey: "k"})
	err := c.Test(context.Background())
	if !errors.Is(err, mailbox.ErrGmailServiceAccount) {
		t.Fatalf("got %v", err)
	}
}

func TestSubjectMustMatchDomain(t *testing.T) {
	t.Parallel()
	c := New("acme.com", "ada@other.com", Key{ClientEmail: "sa@x", PrivateKey: "k"})
	if err := c.Test(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}
