import { Suspense } from "react";
import { EvidenceLibrary } from "@/components/evidence-library";
import { PageBody, PageHeader } from "@/components/page-header";
import { getEvidence } from "@/lib/api";

export const instant = false;

async function EvidenceBody() {
  const data = await getEvidence();
  return (
    <main className="flex min-h-0 flex-1 flex-col">
      <PageHeader title="Evidence" />
      <PageBody>
        <EvidenceLibrary items={data?.evidence ?? []} />
      </PageBody>
    </main>
  );
}

export default function EvidencePage() {
  return (
    <Suspense
      fallback={
        <main className="font-body-md text-muted-foreground">Loading…</main>
      }
    >
      <EvidenceBody />
    </Suspense>
  );
}
