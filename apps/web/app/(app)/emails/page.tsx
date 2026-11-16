import { Suspense } from "react";
import { SavedEmails } from "@/components/saved-emails";
import { PageLoading } from "@/components/page-header";
import { getSavedEmails } from "@/lib/api";

export const instant = false;

async function EmailsBody({
  searchParams,
}: {
  searchParams: Promise<{ template?: string }>;
}) {
  const sp = await searchParams;
  const data = await getSavedEmails();
  return (
    <main className="flex min-h-0 flex-1 flex-col">
      <SavedEmails emails={data?.templates ?? []} template={sp.template ?? null} />
    </main>
  );
}

export default function EmailsPage({
  searchParams,
}: {
  searchParams: Promise<{ template?: string }>;
}) {
  return (
    <Suspense fallback={<PageLoading />}>
      <EmailsBody searchParams={searchParams} />
    </Suspense>
  );
}
