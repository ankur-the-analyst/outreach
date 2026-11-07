import { Suspense } from "react";
import { ConnectMailboxForm } from "@/components/connect-mailbox-form";
import { MailboxList } from "@/components/mailbox-list";
import { PageBody, PageHeader, PageLoading } from "@/components/page-header";
import { getMailboxes } from "@/lib/api";

export const instant = false;

async function MailboxesBody() {
  const data = await getMailboxes();
  return (
    <main className="flex min-h-0 flex-1 flex-col">
      <PageHeader title="Mailboxes" />
      <PageBody>
        <MailboxList mailboxes={data?.mailboxes ?? []} />
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
