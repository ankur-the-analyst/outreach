import { Suspense } from "react";
import { PageBody, PageHeader, PageLoading } from "@/components/page-header";
import { JobBoard } from "@/components/job-board";
import { getJobs, getProfiles } from "@/lib/api";

export const instant = false;

async function JobsBody() {
  const [jobs, profiles] = await Promise.all([getJobs(), getProfiles()]);
  return (
    <main className="flex min-h-0 flex-1 flex-col">
      <PageHeader title="Jobs" />
      <PageBody>
        <JobBoard
          jobs={jobs?.jobs ?? []}
          profiles={profiles?.profiles ?? []}
        />
      </PageBody>
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
