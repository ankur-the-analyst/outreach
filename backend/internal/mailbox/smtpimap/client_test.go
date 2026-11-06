package smtpimap

import (
	"context"
	"testing"

	"outreach/internal/mime"
)

func TestSendRejectsEmptyMessage(t *testing.T) {
	t.Parallel()
	c := New(Config{SMTPHost: "127.0.0.1", SMTPPort: 1025})
	_, err := c.Send(context.Background(), &mime.Built{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAppendSentNoIMAPIsNoop(t *testing.T) {
	t.Parallel()
	c := New(Config{SMTPHost: "127.0.0.1"})
	if err := c.AppendSent(context.Background(), &mime.Built{Bytes: []byte("x")}); err != nil {
		t.Fatal(err)
	}
}

func TestFindSentNoIMAPIsFalse(t *testing.T) {
	t.Parallel()
	c := New(Config{})
	ok, err := c.FindSent(context.Background(), "<id@x>")
	if err != nil || ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}
