package mailbox

import (
	"context"
	"errors"

	"outreach/internal/mime"
)

var ErrNotConfigured = errors.New("oauth transport not configured")

type SendResult struct {
	ProviderID string
}

type Transport interface {
	Test(ctx context.Context) error
	Send(ctx context.Context, m *mime.Built) (SendResult, error)
	AppendSent(ctx context.Context, m *mime.Built) error
	FindSent(ctx context.Context, messageID string) (bool, error)
}
