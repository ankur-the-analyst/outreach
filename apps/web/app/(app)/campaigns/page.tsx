import { Suspense } from "react";
import { CampaignBoard } from "@/components/campaign-board";
import { getCampaigns, getContacts, getMailboxes } from "@/lib/api";

export const instant = false;

async function CampaignsBody() {
  const [campaigns, contacts, mailboxes] = await Promise.all([
    getCampaigns(),
    getContacts(),
    getMailboxes(),
  ]);
  return (
    <main>
      <h1 className="font-headline-xl text-foreground">Campaigns</h1>
      <p className="mt-2 mb-6 font-body-md text-muted-foreground">
        Pick a type, add people with a note, then start. Cold and freelance
        need your name and postal address first.
      </p>
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
    <Suspense
      fallback={
        <main className="font-body-md text-muted-foreground">Loading…</main>
      }
    >
      <CampaignsBody />
    </Suspense>
  );
}
