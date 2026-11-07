package campaign

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
)

//go:embed packs/*.json
var packFS embed.FS

type PackStep struct {
	Position int    `json:"position"`
	Kind     string `json:"kind"`
	Content  string `json:"content,omitempty"`
	WaitDays int    `json:"wait_days,omitempty"`
}

type Pack struct {
	ID         string     `json:"id"`
	Intent     string     `json:"intent"`
	Commercial bool       `json:"commercial"`
	Name       string     `json:"name"`
	Steps      []PackStep `json:"steps"`
}

func LoadPacks() (map[string]Pack, error) {
	out := map[string]Pack{}
	err := fs.WalkDir(packFS, "packs", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		b, err := packFS.ReadFile(path)
		if err != nil {
			return err
		}
		var p Pack
		if err := json.Unmarshal(b, &p); err != nil {
			return fmt.Errorf("campaign: pack %s: %w", path, err)
		}
		if p.ID == "" {
			return fmt.Errorf("campaign: pack %s: id required", path)
		}
		out[p.ID] = p
		return nil
	})
	return out, err
}

func (p Pack) CommercialStart() bool {
	if p.Commercial {
		return true
	}
	return p.Intent == "cold" || p.Intent == "freelance"
}
