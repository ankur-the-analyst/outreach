package workspace

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"outreach/internal/auth"
	"outreach/internal/db"
)

func TestPostgresInviteAndKillSwitch(t *testing.T) {
	url := os.Getenv("DATABASE_APP_URL")
	if url == "" {
		t.Skip("DATABASE_APP_URL not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := db.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	ownerID, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	memberID, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	wsID, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = pool.InTx(c, db.GUC{WorkspaceID: wsID}, func(ctx context.Context, q *db.Queries) error {
			_ = q.DeleteSessionsForWorkspace(ctx, wsID)
			_ = q.DeleteMembershipsForWorkspace(ctx, wsID)
			_ = q.DeleteWorkspace(ctx, wsID)
			return nil
		})
		_ = pool.Queries().DeleteUser(c, ownerID)
		_ = pool.Queries().DeleteUser(c, memberID)
		pool.Close()
	})

	err = pool.InTx(ctx, db.GUC{WorkspaceID: wsID, UserID: ownerID}, func(ctx context.Context, q *db.Queries) error {
		if err := q.InsertWorkspace(ctx, db.InsertWorkspaceParams{ID: wsID, Name: "ws-m14"}); err != nil {
			return err
		}
		if err := q.InsertUser(ctx, db.InsertUserParams{
			ID: ownerID, Email: "owner-" + ownerID.String() + "@ex.co", Name: pgtype.Text{String: "Owner", Valid: true},
		}); err != nil {
			return err
		}
		if err := q.InsertUser(ctx, db.InsertUserParams{
			ID: memberID, Email: "member-" + memberID.String() + "@ex.co", Name: pgtype.Text{String: "Member", Valid: true},
		}); err != nil {
			return err
		}
		return q.InsertMembership(ctx, db.InsertMembershipParams{WorkspaceID: wsID, UserID: ownerID, Role: RoleOwner})
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	api := New(NewPostgres(pool))
	ownerSess := auth.Session{UserID: ownerID, WorkspaceID: wsID}
	memberSess := auth.Session{UserID: memberID, WorkspaceID: wsID}

	invite := sessionReq(http.MethodPost, "/api/v1/workspace/members", `{"email":"member-`+memberID.String()+`@ex.co","role":"member"}`, ownerSess)
	rec := httptest.NewRecorder()
	api.Invite(rec, invite)
	if rec.Code != http.StatusCreated {
		t.Fatalf("invite: %d %s", rec.Code, rec.Body.Bytes())
	}

	pause := sessionReq(http.MethodPut, "/api/v1/workspace/kill-switch", `{"paused":true}`, ownerSess)
	rec = httptest.NewRecorder()
	api.PutKillSwitch(rec, pause)
	if rec.Code != http.StatusOK {
		t.Fatalf("owner pause: %d %s", rec.Code, rec.Body.Bytes())
	}

	deny := sessionReq(http.MethodDelete, "/api/v1/workspace", "", memberSess)
	rec = httptest.NewRecorder()
	api.Delete(rec, deny)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("member delete: %d %s", rec.Code, rec.Body.Bytes())
	}

	got, err := NewPostgres(pool).Get(ctx, wsID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.SendsPaused {
		t.Fatal("kill switch not set")
	}
}
