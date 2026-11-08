import { Suspense } from "react";
import { ConnectMailboxForm } from "@/components/connect-mailbox-form";
import { MailboxGroups } from "@/components/mailbox-groups";
import { MailboxList } from "@/components/mailbox-list";
import { MailboxRamps } from "@/components/mailbox-ramps";
import { WorkspaceKeys } from "@/components/workspace-keys";
import { PageBody, PageHeader, PageLoading } from "@/components/page-header";
import {
  getMailboxes,
  getMailboxPools,
  getMailboxUsageByID,
  getSendingDomains,
  getWarmupProfiles,
  getWorkspaceKeys,
} from "@/lib/api";

export const instant = false;

async function MailboxesBody() {
  const data = await getMailboxes();
  const mailboxes = data?.mailboxes ?? [];
  const [usageByMailbox, keys, domains, pools, ramps] = await Promise.all([
    getMailboxUsageByID(mailboxes),
    getWorkspaceKeys(),
    getSendingDomains(),
    getMailboxPools(),
    getWarmupProfiles(),
  ]);
  return (
    <main className="flex min-h-0 flex-1 flex-col">
      <PageHeader title="Mailboxes" />
      <PageBody>
        <MailboxList
          mailboxes={mailboxes}
          usageByMailbox={usageByMailbox}
          domains={domains?.domains ?? []}
        />
        <MailboxGroups mailboxes={mailboxes} pools={pools?.pools ?? []} />
        <MailboxRamps profiles={ramps?.profiles ?? []} />
        <WorkspaceKeys keys={keys?.service_accounts ?? []} />
        <ConnectMailboxForm />
      </PageBody>
    </main>
  );
}

export default function MailboxesPage() {
  return (
    <Suspense fallback={<PageLoading />}>
      <MailboxesBody />
    </Suspense>
  );
}
