package worker

import "testing"

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
