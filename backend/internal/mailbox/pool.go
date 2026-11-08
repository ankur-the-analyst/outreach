package mailbox

import (
	"github.com/google/uuid"
	"outreach/internal/db"
)

type PoolMailbox struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}

type Pool struct {
	ID        uuid.UUID     `json:"id"`
	Name      string        `json:"name"`
	Mailboxes []PoolMailbox `json:"mailboxes"`
}

func assemblePools(rows []db.MailboxPool, members []db.ListMailboxPoolMembersRow) []Pool {
	out := make([]Pool, len(rows))
	idx := make(map[uuid.UUID]int, len(rows))
	for i, row := range rows {
		out[i] = Pool{ID: row.ID, Name: row.Name, Mailboxes: []PoolMailbox{}}
		idx[row.ID] = i
	}
	for _, m := range members {
		i, ok := idx[m.PoolID]
		if !ok {
			continue
		}
		out[i].Mailboxes = append(out[i].Mailboxes, PoolMailbox{ID: m.MailboxID, Email: m.Email})
	}
	return out
}
