import { Suspense } from "react";
import { Inbox } from "@/components/inbox";
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
    <main>
      <h1 className="font-headline-xl text-foreground">Inbox</h1>
      <p className="mt-2 mb-6 font-body-md text-muted-foreground">
        Open a thread to read it. Unread clears when you open it.
      </p>
      <Inbox
        mailboxes={mailboxes}
        mailboxID={mailboxID}
        threads={data?.threads ?? []}
        threadID={sp.thread ?? ""}
      />
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
