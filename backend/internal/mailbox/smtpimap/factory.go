package smtpimap

import (
	"context"

	"outreach/internal/mailbox"
)

func Factory(acc mailbox.Account) (mailbox.Operations, error) {
	if acc.SMTPHost == "" {
		return nil, mailbox.ErrNoSMTP
	}
	return New(Config{
		SMTPHost: acc.SMTPHost,
		SMTPPort: acc.SMTPPort,
		IMAPHost: acc.IMAPHost,
		IMAPPort: acc.IMAPPort,
		Username: acc.Username,
		Password: string(acc.Secret),
	}), nil
}

func (c *Client) Fetch(context.Context, mailbox.FetchCursor) (mailbox.FetchResult, error) {
	return mailbox.FetchResult{}, mailbox.ErrInboxFallback
}

var _ mailbox.Operations = (*Client)(nil)
