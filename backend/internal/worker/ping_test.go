package worker

import "testing"

func TestSyncArgsKindAndUnique(t *testing.T) {
	t.Parallel()
	var a SyncArgs
	if a.Kind() != "mailbox.sync" {
		t.Fatalf("kind %q", a.Kind())
	}
	if !a.InsertOpts().UniqueOpts.ByArgs {
		t.Fatal("expected unique by args")
	}
}

func TestReconcileArgsKindAndUnique(t *testing.T) {
	t.Parallel()
	var a ReconcileArgs
	if a.Kind() != "ledger.reconcile" {
		t.Fatalf("kind %q", a.Kind())
	}
	if !a.InsertOpts().UniqueOpts.ByArgs {
		t.Fatal("expected unique by args")
	}
}

func TestPingArgsKindAndUnique(t *testing.T) {
	t.Parallel()
	var a PingArgs
	if a.Kind() != "workspace.ping" {
		t.Fatalf("kind %q", a.Kind())
	}
	if !a.InsertOpts().UniqueOpts.ByArgs {
		t.Fatal("expected unique by args")
	}
}
