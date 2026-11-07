import { Suspense } from "react";
import { OverviewBoard } from "@/components/overview-board";
import { PageLoading } from "@/components/page-header";
import { getCampaigns, getDrafts, getMailboxes } from "@/lib/api";

export const instant = false;

async function OverviewBody() {
  const [drafts, mailboxes, campaigns] = await Promise.all([
    getDrafts(),
    getMailboxes(),
    getCampaigns(),
  ]);
  return (
    <main className="flex min-h-0 flex-1 flex-col">
      <OverviewBoard
        drafts={drafts?.drafts ?? []}
        mailboxes={mailboxes?.mailboxes ?? []}
        campaigns={campaigns?.campaigns ?? []}
      />
    </main>
  );
}

export default function OverviewPage() {
  return (
    <Suspense fallback={<PageLoading />}>
      <OverviewBody />
    </Suspense>
  );
}
