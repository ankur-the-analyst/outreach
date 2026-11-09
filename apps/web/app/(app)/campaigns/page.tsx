import { Suspense } from "react";
import { CampaignBoard } from "@/components/campaign-board";
import { PageLoading } from "@/components/page-header";
import {
  getCampaignPeople,
  getCampaigns,
  getContacts,
  getJobs,
  getMailboxPools,
  getMailboxes,
  getProfiles,
} from "@/lib/api";

export const instant = false;

async function CampaignsBody() {
  const [campaigns, contacts, mailboxes, pools, jobs, profiles] =
    await Promise.all([
      getCampaigns(),
      getContacts(),
      getMailboxes(),
      getMailboxPools(),
      getJobs(),
      getProfiles(),
    ]);
  const list = campaigns?.campaigns ?? [];
  const peopleEntries = await Promise.all(
    list.map(async (c) => [c.id, await getCampaignPeople(c.id)] as const),
  );
  const peopleByCampaign = Object.fromEntries(peopleEntries);
  return (
    <main className="flex min-h-0 flex-1 flex-col">
      <CampaignBoard
        campaigns={list}
        contacts={contacts?.contacts ?? []}
        mailboxes={mailboxes?.mailboxes ?? []}
        pools={pools?.pools ?? []}
        jobs={jobs?.jobs ?? []}
        profiles={profiles?.profiles ?? []}
        peopleByCampaign={peopleByCampaign}
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
