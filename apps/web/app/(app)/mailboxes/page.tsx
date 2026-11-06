import { Suspense } from "react";
import { ConnectMailboxForm } from "@/components/connect-mailbox-form";
import { MailboxList } from "@/components/mailbox-list";
import { getMailboxes } from "@/lib/api";

export const instant = false;

async function MailboxesBody() {
  const data = await getMailboxes();
  return (
    <main>
      <h1 className="font-headline-xl text-foreground">Mailboxes</h1>
      <p className="mt-2 mb-6 font-body-md text-muted-foreground">
        SMTP mailboxes in this workspace. Test talks to the stored host.
      </p>
      <MailboxList mailboxes={data?.mailboxes ?? []} />
      <ConnectMailboxForm />
    </main>
  );
}

export default function MailboxesPage() {
  return (
    <Suspense
      fallback={
        <main className="font-body-md text-muted-foreground">Loading…</main>
      }
    >
      <MailboxesBody />
    </Suspense>
  );
}
