import { Suspense } from "react";
import { ConnectMailboxForm } from "@/components/connect-mailbox-form";
import { MailboxList } from "@/components/mailbox-list";
import { PageBody, PageHeader, PageLoading } from "@/components/page-header";
import { getMailboxes, getMailboxUsageByID } from "@/lib/api";

export const instant = false;

async function MailboxesBody() {
  const data = await getMailboxes();
  const mailboxes = data?.mailboxes ?? [];
  const usageByMailbox = await getMailboxUsageByID(mailboxes);
  return (
    <main className="flex min-h-0 flex-1 flex-col">
      <PageHeader title="Mailboxes" />
      <PageBody>
        <MailboxList mailboxes={mailboxes} usageByMailbox={usageByMailbox} />
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
