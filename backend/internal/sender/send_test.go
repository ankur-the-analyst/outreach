package sender

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"outreach/internal/crypto"
	"outreach/internal/db"
	"outreach/internal/mailbox"
	"outreach/internal/mime"
	"outreach/internal/pacing"
)

type stubTransport struct {
	sendErr   error
	appendErr error
	last      *mime.Built
}

func (s *stubTransport) Test(context.Context) error { return nil }
func (s *stubTransport) Send(_ context.Context, m *mime.Built) (mailbox.SendResult, error) {
	s.last = m
	return mailbox.SendResult{}, s.sendErr
}
func (s *stubTransport) AppendSent(context.Context, *mime.Built) error  { return s.appendErr }
func (s *stubTransport) FindSent(context.Context, string) (bool, error) { return false, nil }

type stubDialer struct{ tr mailbox.Transport }

func (s stubDialer) Dial(context.Context, uuid.UUID, uuid.UUID) (mailbox.Transport, string, error) {
	return s.tr, "dev@example.test", nil
}

func TestAppendSentFailureStillSent(t *testing.T) {
	url := os.Getenv("DATABASE_APP_URL")
	if url == "" {
		t.Skip("DATABASE_APP_URL not set")
	}
	keys, err := crypto.NewKeyring("1:0000000000000000000000000000000000000000000000000000000000000001", 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := db.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	wsID, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = pool.InTx(c, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
			_ = q.DeleteAttemptsForWorkspace(ctx, wsID)
			_ = q.DeleteLedgerForWorkspace(ctx, wsID)
			_ = q.DeleteMailboxesForWorkspace(ctx, wsID)
			_ = q.DeleteSecretsForWorkspace(ctx, wsID)
			_ = q.DeleteWarmupProfilesForWorkspace(ctx, wsID)
			_ = q.DeleteSendingDomainsForWorkspace(ctx, wsID)
			_ = q.DeleteWorkspace(ctx, wsID)
			return nil
		})
		pool.Close()
	})
	err = pool.InTx(ctx, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
		if err := q.InsertWorkspace(ctx, db.InsertWorkspaceParams{ID: wsID, Name: "send-m34"}); err != nil {
			return err
		}
		return pacing.SeedPresets(ctx, q, wsID)
	})
	if err != nil {
		t.Fatal(err)
	}
	mb, err := mailbox.NewPostgres(pool, keys).Connect(ctx, wsID, mailbox.ConnectInput{
		Email: "dev@example.test", AuthKind: "smtp_imap", Timezone: "UTC",
		SMTPHost: "127.0.0.1", SMTPPort: 1025,
	})
	if err != nil {
		t.Fatal(err)
	}
	stub := &stubTransport{appendErr: errors.New("imap down")}
	led := NewPostgres(pool, time.Now, time.Minute).WithDialer(stubDialer{tr: stub})
	held, err := led.Hold(ctx, wsID, HoldInput{MailboxID: mb.ID, RFCMessageID: "<stable@outreach.test>"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := led.Advance(ctx, wsID, held.ID, pacing.StateReserved); err != nil {
		t.Fatal(err)
	}
	sent, err := led.Send(ctx, wsID, held.ID, SendInput{To: []string{"sink@outreach.test"}, Subject: "x", Text: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if sent.State != pacing.StateSent {
		t.Fatalf("%s", sent.State)
	}
	if stub.last == nil || stub.last.MessageID != "<stable@outreach.test>" {
		t.Fatalf("%+v", stub.last)
	}
}

func TestMailpitOneOffSend(t *testing.T) {
	smtpAddr := os.Getenv("MAILPIT_SMTP")
	api := os.Getenv("MAILPIT_API")
	url := os.Getenv("DATABASE_APP_URL")
	if smtpAddr == "" || api == "" || url == "" {
		t.Skip("MAILPIT_SMTP, MAILPIT_API, or DATABASE_APP_URL not set")
	}
	host, portStr, err := net.SplitHostPort(smtpAddr)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatal(err)
	}
	keys, err := crypto.NewKeyring("1:0000000000000000000000000000000000000000000000000000000000000001", 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := db.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	wsID, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = pool.InTx(c, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
			_ = q.DeleteAttemptsForWorkspace(ctx, wsID)
			_ = q.DeleteLedgerForWorkspace(ctx, wsID)
			_ = q.DeleteMailboxesForWorkspace(ctx, wsID)
			_ = q.DeleteSecretsForWorkspace(ctx, wsID)
			_ = q.DeleteWarmupProfilesForWorkspace(ctx, wsID)
			_ = q.DeleteSendingDomainsForWorkspace(ctx, wsID)
			_ = q.DeleteWorkspace(ctx, wsID)
			return nil
		})
		pool.Close()
	})
	err = pool.InTx(ctx, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
		if err := q.InsertWorkspace(ctx, db.InsertWorkspaceParams{ID: wsID, Name: "mailpit-m34"}); err != nil {
			return err
		}
		return pacing.SeedPresets(ctx, q, wsID)
	})
	if err != nil {
		t.Fatal(err)
	}
	mb, err := mailbox.NewPostgres(pool, keys).Connect(ctx, wsID, mailbox.ConnectInput{
		Email: "dev@example.test", AuthKind: "smtp_imap", Timezone: "UTC",
		SMTPHost: host, SMTPPort: port,
	})
	if err != nil {
		t.Fatal(err)
	}
	id, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	msgid := "<" + id.String() + "@outreach.test>"
	led := NewPostgres(pool, time.Now, time.Minute).WithDialer(NewMailboxDialer(pool, keys))
	held, err := led.Hold(ctx, wsID, HoldInput{MailboxID: mb.ID, RFCMessageID: msgid})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := led.Advance(ctx, wsID, held.ID, pacing.StateReserved); err != nil {
		t.Fatal(err)
	}
	sent, err := led.Send(ctx, wsID, held.ID, SendInput{
		To: []string{"sink@outreach.test"}, Subject: "m3.4", Text: "hello",
	})
	if err != nil {
		t.Fatal(err)
	}
	if sent.State != pacing.StateSent || sent.RFCMessageID != msgid {
		t.Fatalf("%+v", sent)
	}
	want := id.String() + "@outreach.test"
	deadline := time.Now().Add(5 * time.Second)
	for {
		ok, err := mailpitHas(ctx, api, want)
		if err == nil && ok {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("mailpit missing %s: %v", want, err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func mailpitHas(ctx context.Context, api, messageID string) (bool, error) {
	u := strings.TrimRight(api, "/") + "/api/v1/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return false, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()
	var body struct {
		Messages []struct {
			MessageID string `json:"MessageID"`
		} `json:"messages"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return false, err
	}
	for _, m := range body.Messages {
		if m.MessageID == messageID {
			return true, nil
		}
	}
	return false, nil
}
