package evidence

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"outreach/internal/ai"
	"outreach/internal/db"
)

func TestProposeConfirmRetrieveAndCite(t *testing.T) {
	url := os.Getenv("DATABASE_APP_URL")
	if url == "" {
		t.Skip("DATABASE_APP_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := db.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	wsID, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		c, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		_ = pool.WithGUC(c, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
			_ = q.DeleteEvidenceForWorkspace(ctx, wsID)
			_ = q.DeleteProfilesForWorkspace(ctx, wsID)
			_ = q.DeleteWorkspace(ctx, wsID)
			return nil
		})
		pool.Close()
	})
	err = pool.WithGUC(ctx, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
		return q.InsertWorkspace(ctx, db.InsertWorkspaceParams{ID: wsID, Name: "m52"})
	})
	if err != nil {
		t.Fatal(err)
	}
	store := NewPostgres(pool, &ai.Fake{Model: "fake-8", Dim: Dim})
	unconf, err := store.Propose(ctx, wsID, ProposeInput{Text: "Closed 12 deals last year."})
	if err != nil || unconf.Confirmed {
		t.Fatalf("%+v %v", unconf, err)
	}
	_, err = store.Cite(ctx, wsID, []uuid.UUID{unconf.ID})
	if !errors.Is(err, ErrUnconfirmed) {
		t.Fatalf("cite unconfirmed: %v", err)
	}
	conf, err := store.Confirm(ctx, wsID, unconf.ID)
	if err != nil || !conf.Confirmed {
		t.Fatalf("%+v %v", conf, err)
	}
	cited, err := store.Cite(ctx, wsID, []uuid.UUID{conf.ID})
	if err != nil || len(cited) != 1 {
		t.Fatalf("%+v %v", cited, err)
	}
	if err := GateNumbers("Closed 12 deals last year.", cited); err != nil {
		t.Fatal(err)
	}
	if err := GateNumbers("Closed 99 deals last year.", cited); !errors.Is(err, ErrClaim) {
		t.Fatalf("gate 99: %v", err)
	}
	got, err := store.Retrieve(ctx, wsID, nil, "sales")
	if err != nil || len(got) != 1 || got[0].ID != conf.ID {
		t.Fatalf("retrieve %+v %v", got, err)
	}

	adaID, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	otherID, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	err = pool.WithGUC(ctx, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
		if _, err := q.InsertProfile(ctx, db.InsertProfileParams{WorkspaceID: wsID, ID: adaID, Name: "Ada", About: pgtype.Text{}}); err != nil {
			return err
		}
		_, err := q.InsertProfile(ctx, db.InsertProfileParams{WorkspaceID: wsID, ID: otherID, Name: "Other", About: pgtype.Text{}})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	adaItem, err := store.Propose(ctx, wsID, ProposeInput{Text: "Ada shipped v2.", ProfileID: &adaID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Confirm(ctx, wsID, adaItem.ID); err != nil {
		t.Fatal(err)
	}
	otherItem, err := store.Propose(ctx, wsID, ProposeInput{Text: "Other closed a seed round.", ProfileID: &otherID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Confirm(ctx, wsID, otherItem.ID); err != nil {
		t.Fatal(err)
	}
	adaSet, err := store.Retrieve(ctx, wsID, &adaID, "sales")
	if err != nil || len(adaSet) != 2 {
		t.Fatalf("ada retrieve %+v %v", adaSet, err)
	}
	for _, it := range adaSet {
		if it.ID == otherItem.ID {
			t.Fatal("other profile leaked into ada retrieve")
		}
	}
}
