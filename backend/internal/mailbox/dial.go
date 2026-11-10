package mailbox

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"outreach/internal/crypto"
	"outreach/internal/db"
)

type SecretOpener interface {
	Open(s crypto.Sealed) ([]byte, error)
}

type Dialer struct {
	pool *db.Pool
	keys SecretOpener
	reg  *Registry
}

func NewDialer(pool *db.Pool, keys SecretOpener, reg *Registry) Dialer {
	return Dialer{pool: pool, keys: keys, reg: reg}
}

func (d Dialer) Open(ctx context.Context, workspaceID, mailboxID uuid.UUID) (Operations, string, error) {
	var from string
	var ops Operations
	err := d.pool.WithGUC(ctx, db.GUC{WorkspaceID: workspaceID}, func(ctx context.Context, q *db.Queries) error {
		row, err := q.GetMailboxDial(ctx, db.GetMailboxDialParams{WorkspaceID: workspaceID, ID: mailboxID})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		from = row.Email
		acc := Account{
			Email:    row.Email,
			AuthKind: row.AuthKind,
			Provider: row.Provider,
			SMTPHost: row.SmtpHost.String,
			SMTPPort: int(row.SmtpPort.Int32),
			IMAPHost: row.ImapHost.String,
			IMAPPort: int(row.ImapPort.Int32),
			Username: row.Username.String,
		}
		if acc.Username == "" {
			acc.Username = row.Email
		}
		if len(row.WrappedDek) > 0 && d.keys != nil {
			plain, err := d.keys.Open(crypto.Sealed{
				KeyVersion: int(row.KeyVersion.Int32),
				WrappedDEK: row.WrappedDek,
				Nonce:      row.Nonce,
				Ciphertext: row.Ciphertext,
			})
			if err != nil {
				return fmt.Errorf("mailbox: secret: %w", err)
			}
			acc.Secret = plain
		}
		opened, err := d.reg.Open(acc)
		if err != nil {
			return err
		}
		ops = opened
		return nil
	})
	return ops, from, err
}

func (d Dialer) Dial(ctx context.Context, workspaceID, mailboxID uuid.UUID) (Transport, string, error) {
	return d.Open(ctx, workspaceID, mailboxID)
}
