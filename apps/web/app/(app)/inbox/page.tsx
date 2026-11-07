import { Suspense } from "react";
import { Inbox } from "@/components/inbox";
import { PageBody, PageHeader } from "@/components/page-header";
import { getMailboxes, getThreads } from "@/lib/api";

export const instant = false;

async function InboxBody({
  searchParams,
}: {
  searchParams: Promise<{ mailbox?: string; thread?: string }>;
}) {
  const sp = await searchParams;
  const boxes = await getMailboxes();
  const mailboxes = boxes?.mailboxes ?? [];
  const mailboxID =
    mailboxes.find((m) => m.id === sp.mailbox)?.id ?? mailboxes[0]?.id ?? "";
  const data = mailboxID ? await getThreads(mailboxID) : null;
  return (
    <main className="flex min-h-0 flex-1 flex-col">
      <PageHeader title="Inbox" />
      <PageBody>
        <Inbox
          mailboxes={mailboxes}
          mailboxID={mailboxID}
          threads={data?.threads ?? []}
          threadID={sp.thread ?? ""}
        />
      </PageBody>
    </main>
  );
}

export default function InboxPage({
  searchParams,
}: {
  searchParams: Promise<{ mailbox?: string; thread?: string }>;
}) {
  return (
    <Suspense
      fallback={
        <main className="font-body-md text-muted-foreground">Loading…</main>
      }
    >
      <InboxBody searchParams={searchParams} />
    </Suspense>
  );
}
