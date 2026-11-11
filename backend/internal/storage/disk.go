package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var ErrBadKey = errors.New("storage: bad key")

type Disk struct {
	root string
}

func NewDisk(root string) (*Disk, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, errors.New("storage: path required")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("storage: path: %w", err)
	}
	if err := os.MkdirAll(abs, 0o700); err != nil {
		return nil, fmt.Errorf("storage: mkdir: %w", err)
	}
	return &Disk{root: abs}, nil
}

func (d *Disk) Put(ctx context.Context, key, _ string, data []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := d.resolve(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("storage: mkdir: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("storage: write: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("storage: rename: %w", err)
	}
	return nil
}

func (d *Disk) Get(ctx context.Context, key string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	path, err := d.resolve(key)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("storage: read: %w", err)
	}
	return b, nil
}

func (d *Disk) resolve(key string) (string, error) {
	key = filepath.ToSlash(strings.TrimSpace(key))
	if key == "" || strings.HasPrefix(key, "/") {
		return "", ErrBadKey
	}
	for _, p := range strings.Split(key, "/") {
		if p == "" || p == "." || p == ".." {
			return "", ErrBadKey
		}
	}
	return filepath.Join(d.root, filepath.FromSlash(key)), nil
}

func Open(driver, path string) (Store, error) {
	switch strings.TrimSpace(driver) {
	case "", "disk":
		return NewDisk(path)
	case "memory":
		return NewMemory(), nil
	default:
		return nil, fmt.Errorf("storage: unknown driver %q", driver)
	}
}
