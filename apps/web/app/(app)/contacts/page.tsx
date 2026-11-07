import { Suspense } from "react";
import { ContactBook } from "@/components/contact-book";
import { PageBody, PageHeader } from "@/components/page-header";
import { getContacts } from "@/lib/api";

export const instant = false;

async function ContactsBody() {
  const data = await getContacts();
  return (
    <main className="flex min-h-0 flex-1 flex-col">
      <PageHeader title="Contacts" />
      <PageBody>
        <ContactBook contacts={data?.contacts ?? []} />
      </PageBody>
    </main>
  );
}

export default function ContactsPage() {
  return (
    <Suspense
      fallback={
        <main className="font-body-md text-muted-foreground">Loading…</main>
      }
    >
      <ContactsBody />
    </Suspense>
  );
}
