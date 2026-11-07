import { Suspense } from "react";
import { OneOffDraftForm } from "@/components/one-off-draft-form";
import { ReviewQueue } from "@/components/review-queue";
import { getDrafts, getMailboxes } from "@/lib/api";

export const instant = false;

async function ReviewBody() {
  const [drafts, mailboxes] = await Promise.all([getDrafts(), getMailboxes()]);
  return (
    <main>
      <h1 className="font-headline-xl text-foreground">Review</h1>
      <p className="mt-2 mb-6 font-body-md text-muted-foreground">
        Approve a one-off draft to start a 5-minute hold. Undo in that window
        cancels the send.
      </p>
      <ReviewQueue drafts={drafts?.drafts ?? []} />
      <OneOffDraftForm mailboxes={mailboxes?.mailboxes ?? []} />
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
