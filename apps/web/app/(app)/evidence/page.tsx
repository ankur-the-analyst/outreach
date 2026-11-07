import { Suspense } from "react";
import { EvidenceLibrary } from "@/components/evidence-library";
import { getEvidence } from "@/lib/api";

export const instant = false;

async function EvidenceBody() {
  const data = await getEvidence();
  return (
    <main>
      <h1 className="font-headline-xl text-foreground">Evidence</h1>
      <p className="mt-2 mb-6 font-body-md text-muted-foreground">
        Confirm a claim before compose can cite it. Unconfirmed items stay out
        of facts_used.
      </p>
      <EvidenceLibrary items={data?.evidence ?? []} />
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
