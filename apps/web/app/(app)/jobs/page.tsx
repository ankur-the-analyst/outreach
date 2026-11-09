import { Suspense } from "react";
import { PageLoading } from "@/components/page-header";
import { JobBoard } from "@/components/job-board";
import { getFits, getJobs, getProfiles } from "@/lib/api";

export const instant = false;

async function JobsBody() {
  const [jobs, profiles, fits] = await Promise.all([
    getJobs(),
    getProfiles(),
    getFits(),
  ]);
  return (
    <main className="flex min-h-0 flex-1 flex-col">
      <JobBoard
        jobs={jobs?.jobs ?? []}
        nextCursor={jobs?.next_cursor}
        profiles={profiles?.profiles ?? []}
        fits={fits?.fits ?? []}
      />
    </main>
  );
}

export default function JobsPage() {
  return (
    <Suspense fallback={<PageLoading />}>
      <JobsBody />
    </Suspense>
  );
}
