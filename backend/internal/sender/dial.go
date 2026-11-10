package sender

import (
	"outreach/internal/crypto"
	"outreach/internal/db"
	"outreach/internal/mailbox"
	"outreach/internal/mailbox/smtpimap"
)

type Opener interface {
	Open(s crypto.Sealed) ([]byte, error)
}

func NewMailboxDialer(pool *db.Pool, keys Opener) mailbox.Dialer {
	reg := mailbox.NewRegistry()
	reg.Register("smtp_imap", "", smtpimap.Factory)
	reg.Register("app_password", "", smtpimap.Factory)
	return mailbox.NewDialer(pool, keys, reg)
}
