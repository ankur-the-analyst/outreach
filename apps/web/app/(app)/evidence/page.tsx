import { Suspense } from "react";
import { EvidenceLibrary } from "@/components/evidence-library";
import { PageBody, PageHeader, PageLoading } from "@/components/page-header";
import { getEvidence, getProfiles } from "@/lib/api";

export const instant = false;

async function EvidenceBody() {
  const [data, profiles] = await Promise.all([getEvidence(), getProfiles()]);
  return (
    <main className="flex min-h-0 flex-1 flex-col">
      <PageHeader title="Evidence" />
      <PageBody>
        <EvidenceLibrary
          items={data?.evidence ?? []}
          profiles={profiles?.profiles ?? []}
        />
      </PageBody>
    </main>
  );
}

export default function EvidencePage() {
  return (
    <Suspense fallback={<PageLoading />}>
      <EvidenceBody />
    </Suspense>
  );
}
