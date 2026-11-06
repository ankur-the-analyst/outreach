package pacing

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"outreach/internal/db"
)

//go:embed profiles.json
var bundledJSON []byte

func Bundled() ([]Preset, error) {
	var presets []Preset
	if err := json.Unmarshal(bundledJSON, &presets); err != nil {
		return nil, fmt.Errorf("pacing bundled: %w", err)
	}
	return presets, nil
}

func SeedPresets(ctx context.Context, q *db.Queries, workspaceID uuid.UUID) error {
	presets, err := Bundled()
	if err != nil {
		return err
	}
	for _, p := range presets {
		id, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("pacing seed id: %w", err)
		}
		raw, err := json.Marshal(p.Spec)
		if err != nil {
			return fmt.Errorf("pacing seed spec: %w", err)
		}
		if err := q.InsertWarmupProfile(ctx, db.InsertWarmupProfileParams{
			WorkspaceID: workspaceID,
			ID:          id,
			Name:        p.Name,
			Kind:        p.Kind,
			Spec:        raw,
			Version:     1,
		}); err != nil {
			return fmt.Errorf("pacing seed %s: %w", p.Kind, err)
		}
	}
	return nil
}
