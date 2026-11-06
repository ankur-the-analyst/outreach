export default function OverviewPage() {
  return (
    <main className="rounded-md border border-border bg-card p-6">
      <h1 className="font-headline-xl text-foreground">Overview</h1>
      <p className="mt-2 font-body-md text-muted-foreground">
        Nothing queued yet. Mailboxes and campaigns come in later slices.
      </p>
    </main>
  );
}
