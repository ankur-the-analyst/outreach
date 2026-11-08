import { Suspense } from "react";
import { CampaignBoard } from "@/components/campaign-board";
import { PageLoading } from "@/components/page-header";
import {
  getCampaigns,
  getContacts,
  getJobs,
  getMailboxes,
  getProfiles,
} from "@/lib/api";

export const instant = false;

async function CampaignsBody() {
  const [campaigns, contacts, mailboxes, jobs, profiles] = await Promise.all([
    getCampaigns(),
    getContacts(),
    getMailboxes(),
    getJobs(),
    getProfiles(),
  ]);
  return (
    <main className="flex min-h-0 flex-1 flex-col">
      <CampaignBoard
        campaigns={campaigns?.campaigns ?? []}
        contacts={contacts?.contacts ?? []}
        mailboxes={mailboxes?.mailboxes ?? []}
        jobs={jobs?.jobs ?? []}
        profiles={profiles?.profiles ?? []}
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
