import { Suspense } from "react";
import { ContactBook } from "@/components/contact-book";
import { getContacts } from "@/lib/api";

export const instant = false;

async function ContactsBody() {
  const data = await getContacts();
  return (
    <main>
      <h1 className="font-headline-xl text-foreground">Contacts</h1>
      <p className="mt-2 mb-6 font-body-md text-muted-foreground">
        Record where a person came from. One email per workspace. Preview a
        CSV before you import.
      </p>
      <ContactBook contacts={data?.contacts ?? []} />
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
