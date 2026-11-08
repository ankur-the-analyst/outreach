import { Suspense } from "react";
import { OverviewBoard } from "@/components/overview-board";
import { PageLoading } from "@/components/page-header";
import {
  getCampaigns,
  getDrafts,
  getMailboxes,
  getMailboxUsageByID,
} from "@/lib/api";

export const instant = false;

async function OverviewBody() {
  const [drafts, mailboxes, campaigns] = await Promise.all([
    getDrafts(),
    getMailboxes(),
    getCampaigns(),
  ]);
  const boxes = mailboxes?.mailboxes ?? [];
  const usageByMailbox = await getMailboxUsageByID(boxes);
  return (
    <main className="flex min-h-0 flex-1 flex-col">
      <OverviewBoard
        drafts={drafts?.drafts ?? []}
        mailboxes={boxes}
        campaigns={campaigns?.campaigns ?? []}
        usageByMailbox={usageByMailbox}
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
