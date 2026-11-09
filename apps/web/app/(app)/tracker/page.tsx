import { Suspense } from "react";
import { PageLoading } from "@/components/page-header";
import { TrackerBoard } from "@/components/tracker-board";
import { getTracker } from "@/lib/api";

export const instant = false;

async function TrackerBody() {
  const data = await getTracker();
  return (
    <main className="flex min-h-0 flex-1 flex-col">
      <TrackerBoard rows={data?.rows ?? []} />
    </main>
  );
}

export default function TrackerPage() {
  return (
    <Suspense fallback={<PageLoading />}>
      <TrackerBody />
    </Suspense>
  );
}
