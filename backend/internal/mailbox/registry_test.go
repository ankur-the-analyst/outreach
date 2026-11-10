package mailbox

import (
	"context"
	"errors"
	"testing"

	"outreach/internal/mime"
)

type fakeOps struct{}

func (fakeOps) Test(context.Context) error { return nil }
func (fakeOps) Send(context.Context, *mime.Built) (SendResult, error) {
	return SendResult{ProviderID: "x"}, nil
}
func (fakeOps) AppendSent(context.Context, *mime.Built) error { return nil }
func (fakeOps) FindSent(context.Context, string) (bool, error) {
	return true, nil
}
func (fakeOps) Fetch(context.Context, FetchCursor) (FetchResult, error) {
	return FetchResult{Messages: []Message{{RFCMessageID: "<a@x>"}}}, nil
}

func TestRegistryOpenExactThenWildcard(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	r.Register("oauth", "gmail", func(Account) (Operations, error) { return fakeOps{}, nil })
	r.Register("smtp_imap", "", func(Account) (Operations, error) { return fakeOps{}, nil })

	gmail, err := r.Open(Account{AuthKind: "oauth", Provider: "gmail"})
	if err != nil {
		t.Fatal(err)
	}
	res, err := gmail.Fetch(context.Background(), FetchCursor{})
	if err != nil || len(res.Messages) != 1 {
		t.Fatalf("gmail fetch %+v %v", res, err)
	}

	smtp, err := r.Open(Account{AuthKind: "smtp_imap", Provider: "other"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := smtp.Send(context.Background(), nil); err != nil {
		t.Fatal(err)
	}

	_, err = r.Open(Account{AuthKind: "oauth", Provider: "outlook"})
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("outlook got %v", err)
	}
}
