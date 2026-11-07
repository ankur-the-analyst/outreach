package storage

import (
	"context"
	"errors"
	"sync"
)

var ErrNotFound = errors.New("storage: not found")

type Store interface {
	Put(ctx context.Context, key, contentType string, data []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
}

type Memory struct {
	mu sync.Mutex
	m  map[string][]byte
}

func NewMemory() *Memory {
	return &Memory{m: map[string][]byte{}}
}

func (s *Memory) Put(_ context.Context, key, _ string, data []byte) error {
	if key == "" {
		return errors.New("storage: key required")
	}
	cp := append([]byte(nil), data...)
	s.mu.Lock()
	s.m[key] = cp
	s.mu.Unlock()
	return nil
}

func (s *Memory) Get(_ context.Context, key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.m[key]
	if !ok {
		return nil, ErrNotFound
	}
	return append([]byte(nil), b...), nil
}
