package oauth

import (
	"context"

	"outreach/internal/mailbox"
	"outreach/internal/mime"
)

type Stub struct{}

var _ mailbox.Transport = Stub{}

func (Stub) Test(context.Context) error {
	return mailbox.ErrNotConfigured
}

func (Stub) Send(context.Context, *mime.Built) (mailbox.SendResult, error) {
	return mailbox.SendResult{}, mailbox.ErrNotConfigured
}

func (Stub) AppendSent(context.Context, *mime.Built) error {
	return mailbox.ErrNotConfigured
}

func (Stub) FindSent(context.Context, string) (bool, error) {
	return false, mailbox.ErrNotConfigured
}
