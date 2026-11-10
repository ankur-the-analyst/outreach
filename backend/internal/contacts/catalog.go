package contacts

import (
	"context"

	"github.com/google/uuid"
	"outreach/internal/fields"
)

type Catalog interface {
	Hints(ctx context.Context, workspaceID uuid.UUID, object string) ([]fields.Hint, error)
	AllowedCustom(ctx context.Context, workspaceID uuid.UUID, object string) (map[string]struct{}, error)
}
