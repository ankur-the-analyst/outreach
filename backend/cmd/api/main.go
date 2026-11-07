package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"outreach/internal/auth"
	"outreach/internal/config"
	"outreach/internal/crypto"
	"outreach/internal/db"
	"outreach/internal/health"
	"outreach/internal/mailbox"
	"outreach/internal/mailbox/smtpimap"
	"outreach/internal/sender"
	"outreach/internal/worker"
	"outreach/internal/workspace"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config", "err", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	pool, err := db.Open(ctx, cfg.DatabaseAppURL)
	cancel()
	if err != nil {
		logger.Error("open database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	keys, err := crypto.NewKeyring(cfg.MasterKeys, cfg.MasterKeyActive)
	if err != nil {
		logger.Error("load master keys", "err", err)
		os.Exit(1)
	}

	a := auth.New(auth.NewPostgres(pool))
	wsStore := workspace.NewPostgres(pool)
	ws := workspace.New(wsStore)
	mb := mailbox.New(mailbox.NewPostgres(pool, keys), smtpimap.Probe{})
	jobs, err := worker.NewInsertClient(pool)
	if err != nil {
		logger.Error("river client", "err", err)
		os.Exit(1)
	}
	wrk := worker.New(pool, wsStore, jobs)
	hold := time.Duration(cfg.UndoHoldS) * time.Second
	snd := sender.New(sender.NewPostgres(pool, time.Now, hold))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health.Handler(pool))
	mux.HandleFunc("POST /api/v1/auth/register", a.Register)
	mux.HandleFunc("POST /api/v1/auth/login", a.Login)
	mux.Handle("POST /api/v1/auth/logout", a.Require(http.HandlerFunc(a.Logout)))
	mux.Handle("GET /api/v1/me", a.Require(http.HandlerFunc(a.Me)))
	mux.Handle("GET /api/v1/workspace", a.Require(http.HandlerFunc(ws.Get)))
	mux.Handle("PATCH /api/v1/workspace", a.Require(http.HandlerFunc(ws.Patch)))
	mux.Handle("DELETE /api/v1/workspace", a.Require(http.HandlerFunc(ws.Delete)))
	mux.Handle("PUT /api/v1/workspace/kill-switch", a.Require(http.HandlerFunc(ws.PutKillSwitch)))
	mux.Handle("GET /api/v1/workspace/members", a.Require(http.HandlerFunc(ws.ListMembers)))
	mux.Handle("POST /api/v1/workspace/members", a.Require(http.HandlerFunc(ws.Invite)))
	mux.Handle("GET /api/v1/mailboxes", a.Require(http.HandlerFunc(mb.List)))
	mux.Handle("POST /api/v1/mailboxes", a.Require(http.HandlerFunc(mb.Create)))
	mux.Handle("PATCH /api/v1/mailboxes/{id}", a.Require(http.HandlerFunc(mb.Patch)))
	mux.Handle("POST /api/v1/mailboxes/{id}/pause", a.Require(http.HandlerFunc(mb.Pause)))
	mux.Handle("POST /api/v1/mailboxes/{id}/reconnect", a.Require(http.HandlerFunc(mb.Reconnect)))
	mux.Handle("POST /api/v1/mailboxes/{id}/test", a.Require(http.HandlerFunc(mb.Test)))
	mux.Handle("GET /api/v1/google-service-accounts", a.Require(http.HandlerFunc(mb.ListSA)))
	mux.Handle("POST /api/v1/google-service-accounts", a.Require(http.HandlerFunc(mb.CreateSA)))
	mux.Handle("POST /api/v1/jobs/ping", a.Require(http.HandlerFunc(wrk.Ping)))
	mux.Handle("POST /api/v1/mailboxes/{id}/holds", a.Require(http.HandlerFunc(snd.Hold)))
	mux.Handle("GET /api/v1/mailboxes/{id}/usage", a.Require(http.HandlerFunc(snd.Usage)))
	mux.Handle("POST /api/v1/ledger/{id}/cancel", a.Require(http.HandlerFunc(snd.Cancel)))

	logger.Info("api listening", "addr", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, mux); err != nil {
		logger.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
