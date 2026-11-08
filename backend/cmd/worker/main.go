package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"outreach/internal/campaign"
	"outreach/internal/config"
	"outreach/internal/crypto"
	"outreach/internal/db"
	"outreach/internal/inbound"
	"outreach/internal/sender"
	"outreach/internal/suppression"
	mailsync "outreach/internal/sync"
	"outreach/internal/worker"
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

	hold := time.Duration(cfg.UndoHoldS) * time.Second
	unknown := time.Duration(cfg.UnknownTimeoutS) * time.Second
	led := sender.NewPostgres(pool, time.Now, hold).
		WithDialer(sender.NewMailboxDialer(pool, keys)).
		WithUnknownTimeout(unknown)
	jobs, err := worker.NewInsertClient(pool)
	if err != nil {
		logger.Error("river insert client", "err", err)
		os.Exit(1)
	}
	suppStore := suppression.NewPostgres(pool, keys)
	proc := inbound.New(pool, suppStore, classifyJobs{client: jobs})
	ing := mailsync.NewPostgres(pool, mailsync.Mailpit{Base: cfg.MailpitAPI}).WithInbound(proc)
	packs, err := campaign.LoadPacks()
	if err != nil {
		logger.Error("load packs", "err", err)
		os.Exit(1)
	}
	camp := campaign.NewPostgres(pool, packs)

	client, err := worker.NewWorkerClient(pool, func(w *river.Workers) {
		worker.AddReconcile(w, func(ctx context.Context, a worker.ReconcileArgs) error {
			_, err := led.Reconcile(ctx, a.WorkspaceID, a.LedgerID, sender.SendInput{
				To: a.To, Subject: a.Subject, Text: a.Text, HTML: a.HTML,
			})
			return err
		})
		worker.AddSync(w, func(ctx context.Context, a worker.SyncArgs) error {
			_, err := ing.SyncMailbox(ctx, a.WorkspaceID, a.MailboxID)
			return err
		})
		worker.AddClassify(w, func(ctx context.Context, a worker.ClassifyArgs) error {
			return proc.ClassifyStored(ctx, a.WorkspaceID, a.MessageID)
		})
		worker.AddStep(w, func(ctx context.Context, a worker.StepArgs) error {
			_, err := camp.DraftDueStep(ctx, a.WorkspaceID, a.EnrolmentID, a.StepPosition)
			return err
		})
	})
	if err != nil {
		logger.Error("river client", "err", err)
		os.Exit(1)
	}

	runCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := client.Start(runCtx); err != nil {
		logger.Error("river start", "err", err)
		os.Exit(1)
	}
	logger.Info("worker listening")
	<-runCtx.Done()

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer stopCancel()
	if err := client.Stop(stopCtx); err != nil {
		logger.Error("river stop", "err", err)
		os.Exit(1)
	}
}

type classifyJobs struct {
	client *river.Client[pgx.Tx]
}

func (j classifyJobs) Enqueue(ctx context.Context, workspaceID, messageID uuid.UUID) error {
	return worker.EnqueueClassify(ctx, j.client, workspaceID, messageID)
}
