import { Suspense } from "react";
import { ContactBook } from "@/components/contact-book";
import { PageBody, PageHeader, PageLoading } from "@/components/page-header";
import { getContacts, getNeverEmail } from "@/lib/api";

export const instant = false;

async function ContactsBody() {
  const [data, blocked] = await Promise.all([getContacts(), getNeverEmail()]);
  return (
    <main className="flex min-h-0 flex-1 flex-col">
      <PageHeader title="Contacts" />
      <PageBody>
        <ContactBook
          contacts={data?.contacts ?? []}
          blocked={blocked?.entries ?? []}
        />
      </PageBody>
    </main>
  );
}

export default function ContactsPage() {
  return (
    <Suspense fallback={<PageLoading />}>
      <ContactsBody />
    </Suspense>
  );
}
