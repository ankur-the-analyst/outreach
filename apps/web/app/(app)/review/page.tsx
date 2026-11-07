import { Suspense } from "react";
import { ReviewQueue } from "@/components/review-queue";
import { getDrafts, getMailboxes } from "@/lib/api";

export const instant = false;

async function ReviewBody() {
  const [drafts, mailboxes] = await Promise.all([getDrafts(), getMailboxes()]);
  return (
    <main className="flex min-h-0 flex-1 flex-col">
      <ReviewQueue
        drafts={drafts?.drafts ?? []}
        mailboxes={mailboxes?.mailboxes ?? []}
        filters
      />
    </main>
  );
}

export default function ReviewPage() {
  return (
    <Suspense
      fallback={
        <main className="font-body-md text-muted-foreground">Loading…</main>
      }
    >
      <ReviewBody />
    </Suspense>
  );
}
