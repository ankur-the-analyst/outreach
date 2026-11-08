import { Suspense } from "react";
import { ConnectMailboxForm } from "@/components/connect-mailbox-form";
import { MailboxList } from "@/components/mailbox-list";
import { WorkspaceKeys } from "@/components/workspace-keys";
import { PageBody, PageHeader, PageLoading } from "@/components/page-header";
import {
  getMailboxes,
  getMailboxUsageByID,
  getSendingDomains,
  getWorkspaceKeys,
} from "@/lib/api";

export const instant = false;

async function MailboxesBody() {
  const data = await getMailboxes();
  const mailboxes = data?.mailboxes ?? [];
  const [usageByMailbox, keys, domains] = await Promise.all([
    getMailboxUsageByID(mailboxes),
    getWorkspaceKeys(),
    getSendingDomains(),
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
