package worker

import "testing"

func TestClassifyArgsKindAndUnique(t *testing.T) {
	t.Parallel()
	var a ClassifyArgs
	if a.Kind() != "inbound.classify" {
		t.Fatalf("kind %q", a.Kind())
	}
	if !a.InsertOpts().UniqueOpts.ByArgs {
		t.Fatal("expected unique by args")
	}
}

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

func TestStepArgsKindAndUnique(t *testing.T) {
	t.Parallel()
	var a StepArgs
	if a.Kind() != "campaign.step" {
		t.Fatalf("kind %q", a.Kind())
	}
	if !a.InsertOpts().UniqueOpts.ByArgs {
		t.Fatal("expected unique by args")
	}
}
