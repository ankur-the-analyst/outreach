package campaign

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"outreach/internal/db"
	"outreach/internal/jobsjd"
)

func mustResume(t *testing.T, ctx context.Context, pool *db.Pool, workspaceID uuid.UUID) uuid.UUID {
	t.Helper()
	id, err := jobsjd.PutProfileResume(ctx, pool, workspaceID)
	if err != nil {
		t.Fatal(err)
	}
	return id
}
