import { Suspense } from "react";
import { notFound } from "next/navigation";
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
        item.href !== "/evidence",
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
    <main className="rounded-md border border-border bg-card p-6">
      <h1 className="font-headline-xl text-foreground">{title}</h1>
      <p className="mt-2 font-body-md text-muted-foreground">Not built yet.</p>
    </main>
  );
}

export default function StubPage({
  params,
}: {
  params: Promise<{ section: string }>;
}) {
  return (
    <Suspense
      fallback={
        <main className="rounded-md border border-border bg-card p-6 font-body-md text-muted-foreground">
          Loading…
        </main>
      }
    >
      <StubBody params={params} />
    </Suspense>
  );
}
