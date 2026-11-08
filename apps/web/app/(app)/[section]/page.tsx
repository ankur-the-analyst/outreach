import { Suspense } from "react";
import { notFound } from "next/navigation";
import { PageBody, PageHeader, PageLoading } from "@/components/page-header";
import { navItems, navLabel } from "@/lib/nav";

export const instant = false;

const stubHrefs = new Set(
  navItems
    .filter(
      (item) =>
        item.href !== "/" &&
        item.href !== "/mailboxes" &&
        item.href !== "/review" &&
        item.href !== "/inbox" &&
        item.href !== "/evidence" &&
        item.href !== "/contacts" &&
        item.href !== "/campaigns" &&
        item.href !== "/tracker" &&
        item.href !== "/settings" &&
        item.href !== "/profiles" &&
        item.href !== "/jobs",
    )
    .map((item) => item.href.slice(1)),
);

export function generateStaticParams() {
  return [...stubHrefs].map((section) => ({ section }));
}

async function StubBody({ params }: { params: Promise<{ section: string }> }) {
  const { section } = await params;
  if (!stubHrefs.has(section)) {
    notFound();
  }
  const title = navLabel(`/${section}`);
  return (
    <main className="flex min-h-0 flex-1 flex-col">
      <PageHeader title={title} />
      <PageBody>
        <p className="font-body-md text-muted-foreground">Not built yet.</p>
      </PageBody>
    </main>
  );
}

export default function StubPage({
  params,
}: {
  params: Promise<{ section: string }>;
}) {
  return (
    <Suspense fallback={<PageLoading />}>
      <StubBody params={params} />
    </Suspense>
  );
}
