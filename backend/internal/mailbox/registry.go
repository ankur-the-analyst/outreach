package mailbox

import (
	"fmt"
	"strings"
	"sync"
)

type Account struct {
	Email    string
	AuthKind string
	Provider string
	SMTPHost string
	SMTPPort int
	IMAPHost string
	IMAPPort int
	Username string
	Secret   []byte
}

type Factory func(Account) (Operations, error)

type Registry struct {
	mu sync.RWMutex
	m  map[string]Factory
}

func NewRegistry() *Registry {
	return &Registry{m: make(map[string]Factory)}
}

func (r *Registry) Register(authKind, provider string, f Factory) {
	if r == nil || f == nil || strings.TrimSpace(authKind) == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.m[factoryKey(authKind, provider)] = f
}

func (r *Registry) Open(acc Account) (Operations, error) {
	if r == nil {
		return nil, ErrNotConfigured
	}
	r.mu.RLock()
	f, ok := r.m[factoryKey(acc.AuthKind, acc.Provider)]
	if !ok {
		f, ok = r.m[factoryKey(acc.AuthKind, "")]
	}
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %s/%s", ErrNotConfigured, acc.AuthKind, acc.Provider)
	}
	return f(acc)
}

func factoryKey(authKind, provider string) string {
	return strings.ToLower(strings.TrimSpace(authKind)) + "\x00" + strings.ToLower(strings.TrimSpace(provider))
}
