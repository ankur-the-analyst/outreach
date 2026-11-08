package mailbox

import (
	"testing"

	"github.com/google/uuid"
	"outreach/internal/db"
)

func TestAssemblePoolsAttachesMembers(t *testing.T) {
	t.Parallel()
	a := uuid.MustParse("01700000-0000-7000-8000-00000000000a")
	b := uuid.MustParse("01700000-0000-7000-8000-00000000000b")
	mb := uuid.MustParse("01700000-0000-7000-8000-00000000000c")
	got := assemblePools(
		[]db.MailboxPool{{ID: b, Name: "B"}, {ID: a, Name: "A"}},
		[]db.ListMailboxPoolMembersRow{
			{PoolID: a, MailboxID: mb, Email: "ada@acme.test"},
		},
	)
	if len(got) != 2 || got[0].Name != "B" || len(got[0].Mailboxes) != 0 {
		t.Fatalf("%+v", got)
	}
	if got[1].Name != "A" || len(got[1].Mailboxes) != 1 || got[1].Mailboxes[0].Email != "ada@acme.test" {
		t.Fatalf("%+v", got[1])
	}
}
