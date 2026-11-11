package jobsjd

import (
	"context"

	"github.com/google/uuid"
	"outreach/internal/db"
	"outreach/internal/storage"
)

func PutProfileResume(ctx context.Context, pool *db.Pool, workspaceID uuid.UUID) (uuid.UUID, error) {
	store := NewPostgres(pool, storage.NewMemory(), nil)
	p, err := store.CreateProfile(ctx, workspaceID, "Ada", "")
	if err != nil {
		return uuid.Nil, err
	}
	_, err = store.UploadResume(ctx, workspaceID, p.ID, "Ada.pdf", "application/pdf", "Ada Lovelace", []byte("%PDF-1.4\n%%EOF\n"))
	if err != nil {
		return uuid.Nil, err
	}
	return p.ID, nil
}
