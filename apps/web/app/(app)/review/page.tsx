import { Suspense } from "react";
import { PageLoading } from "@/components/page-header";
import { ReviewQueue } from "@/components/review-queue";
import { getDrafts, getMailboxes, getProfiles } from "@/lib/api";

export const instant = false;

async function ReviewBody() {
  const [drafts, mailboxes, profiles] = await Promise.all([
    getDrafts(),
    getMailboxes(),
    getProfiles(),
  ]);
  return (
    <main className="flex min-h-0 flex-1 flex-col">
      <ReviewQueue
        drafts={drafts?.drafts ?? []}
        mailboxes={mailboxes?.mailboxes ?? []}
        profiles={profiles?.profiles ?? []}
        filters
      />
    </main>
  );
}

export default function ReviewPage() {
  return (
    <Suspense fallback={<PageLoading />}>
      <ReviewBody />
    </Suspense>
  );
}
