package contacts

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestApplyImportSkipsBadAndFileDupes(t *testing.T) {
	t.Parallel()
	var got []string
	created, _, skipped, err := applyImport([]PreviewRow{
		{Email: "ada@acme.com"},
		{Email: "ADA@acme.com"},
		{Email: ""},
		{Email: "not-an-email"},
		{Email: "bob@acme.com"},
	}, func(row PreviewRow) error {
		got = append(got, row.Email)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if created != 2 || skipped != 3 {
		t.Fatalf("created=%d skipped=%d", created, skipped)
	}
	if len(got) != 2 || got[0] != "ada@acme.com" || got[1] != "bob@acme.com" {
		t.Fatalf("%v", got)
	}
}

func TestApplyImportSkipsExisting(t *testing.T) {
	t.Parallel()
	created, _, skipped, err := applyImport([]PreviewRow{
		{Email: "ada@acme.com"},
		{Email: "bob@acme.com"},
	}, func(row PreviewRow) error {
		if row.Email == "ada@acme.com" {
			return ErrDuplicate
		}
		return nil
	})
	if err != nil || created != 1 || skipped != 1 {
		t.Fatalf("created=%d skipped=%d err=%v", created, skipped, err)
	}
}

func TestImportRequiresEmailColumn(t *testing.T) {
	t.Parallel()
	p := NewPostgres(nil)
	_, err := p.Import(t.Context(), uuid.Nil, uuid.Nil, "csv", "Work Email\nada@acme.com\n", nil)
	if err != ErrMapEmail {
		t.Fatalf("%v", err)
	}
}

func TestApplyImportStopsOnOtherError(t *testing.T) {
	t.Parallel()
	boom := errors.New("db down")
	_, _, _, err := applyImport([]PreviewRow{{Email: "ada@acme.com"}}, func(PreviewRow) error {
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("%v", err)
	}
}
