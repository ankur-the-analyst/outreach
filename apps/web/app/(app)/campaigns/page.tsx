import { Suspense } from "react";
import { CampaignBoard } from "@/components/campaign-board";
import { PageLoading } from "@/components/page-header";
import { getCampaigns, getContacts, getMailboxes } from "@/lib/api";

export const instant = false;

async function CampaignsBody() {
  const [campaigns, contacts, mailboxes] = await Promise.all([
    getCampaigns(),
    getContacts(),
    getMailboxes(),
  ]);
  return (
    <main className="flex min-h-0 flex-1 flex-col">
      <CampaignBoard
        campaigns={campaigns?.campaigns ?? []}
        contacts={contacts?.contacts ?? []}
        mailboxes={mailboxes?.mailboxes ?? []}
      />
    </main>
  );
}

export default function CampaignsPage() {
  return (
    <Suspense fallback={<PageLoading />}>
      <CampaignsBody />
    </Suspense>
  );
}
