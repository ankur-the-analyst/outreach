package mailbox

import (
	"context"
	"errors"
	"time"

	"outreach/internal/mime"
)

var (
	ErrNotConfigured = errors.New("oauth transport not configured")
	ErrInboxFallback = errors.New("mailbox: use inbox source")
)

type SendResult struct {
	ProviderID string
}

type Transport interface {
	Test(ctx context.Context) error
	Send(ctx context.Context, m *mime.Built) (SendResult, error)
	AppendSent(ctx context.Context, m *mime.Built) error
	FindSent(ctx context.Context, messageID string) (bool, error)
}

type FetchCursor struct {
	Token string
}

type Message struct {
	RFCMessageID string
	InReplyTo    string
	Subject      string
	From         string
	To           []string
	Text         string
	HTML         string
	At           time.Time
	ProviderID   string
}

type FetchResult struct {
	Messages []Message
	Cursor   FetchCursor
}

type Operations interface {
	Transport
	Fetch(ctx context.Context, cur FetchCursor) (FetchResult, error)
}
