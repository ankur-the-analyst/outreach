import { Suspense } from "react";
import { SavedEmails } from "@/components/saved-emails";
import { PageLoading } from "@/components/page-header";
import { getSavedEmails } from "@/lib/api";

export const instant = false;

async function EmailsBody() {
  const data = await getSavedEmails();
  return (
    <main className="flex min-h-0 flex-1 flex-col">
      <SavedEmails emails={data?.templates ?? []} />
    </main>
  );
}

export default function EmailsPage() {
  return (
    <Suspense fallback={<PageLoading />}>
      <EmailsBody />
    </Suspense>
  );
}
