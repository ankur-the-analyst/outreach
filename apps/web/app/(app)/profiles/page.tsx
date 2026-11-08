import { Suspense } from "react";
import { PageBody, PageHeader, PageLoading } from "@/components/page-header";
import { ProfileBook } from "@/components/profile-book";
import { getProfiles } from "@/lib/api";

export const instant = false;

async function ProfilesBody() {
  const data = await getProfiles();
  return (
    <main className="flex min-h-0 flex-1 flex-col">
      <PageHeader title="Profiles" />
      <PageBody>
        <ProfileBook profiles={data?.profiles ?? []} />
      </PageBody>
    </main>
  );
}

export default function ProfilesPage() {
  return (
    <Suspense fallback={<PageLoading />}>
      <ProfilesBody />
    </Suspense>
  );
}
