package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"outreach/internal/auth"
	"outreach/internal/config"
	"outreach/internal/db"
	"outreach/internal/health"
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

	a := auth.New(auth.NewPostgres(pool))
	ws := workspace.New(workspace.NewPostgres(pool))

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

	logger.Info("api listening", "addr", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, mux); err != nil {
		logger.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
