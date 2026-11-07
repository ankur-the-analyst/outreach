package sender

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"outreach/internal/crypto"
	"outreach/internal/db"
	"outreach/internal/mailbox"
	"outreach/internal/mailbox/smtpimap"
)

type Opener interface {
	Open(s crypto.Sealed) ([]byte, error)
}

type MailboxDialer struct {
	pool *db.Pool
	keys Opener
}

func NewMailboxDialer(pool *db.Pool, keys Opener) MailboxDialer {
	return MailboxDialer{pool: pool, keys: keys}
}

func (d MailboxDialer) Dial(ctx context.Context, workspaceID, mailboxID uuid.UUID) (mailbox.Transport, string, error) {
	var from string
	var tr mailbox.Transport
	err := d.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		row, err := q.GetMailboxDial(ctx, db.GetMailboxDialParams{WorkspaceID: workspaceID, ID: mailboxID})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		from = row.Email
		switch row.AuthKind {
		case "oauth":
			return mailbox.ErrNotConfigured
		case "service_account":
			return mailbox.ErrNotSupported
		}
		if !row.SmtpHost.Valid || row.SmtpHost.String == "" {
			return mailbox.ErrNoSMTP
		}
		cfg := smtpimap.Config{
			SMTPHost: row.SmtpHost.String,
			SMTPPort: int(row.SmtpPort.Int32),
			IMAPHost: row.ImapHost.String,
			IMAPPort: int(row.ImapPort.Int32),
			Username: row.Username.String,
		}
		if cfg.Username == "" {
			cfg.Username = row.Email
		}
		if len(row.WrappedDek) > 0 && d.keys != nil {
			plain, err := d.keys.Open(crypto.Sealed{
				KeyVersion: int(row.KeyVersion.Int32),
				WrappedDEK: row.WrappedDek,
				Nonce:      row.Nonce,
				Ciphertext: row.Ciphertext,
			})
			if err != nil {
				return fmt.Errorf("sender: secret: %w", err)
			}
			cfg.Password = string(plain)
		}
		tr = smtpimap.New(cfg)
		return nil
	})
	return tr, from, err
}
