import type { ReactNode } from "react";
import { Suspense } from "react";
import { connection } from "next/server";
import { redirect } from "next/navigation";
import { LogoutButton } from "@/components/logout-button";
import { PageLoading } from "@/components/page-header";
import { SidebarNav } from "@/components/sidebar-nav";
import { getMe, getWorkspace } from "@/lib/api";

export const instant = false;

async function AppShell({ children }: { children: ReactNode }) {
  await connection();
  const me = await getMe();
  if (!me) {
    redirect("/login");
  }
  const workspace = await getWorkspace();

  return (
    <div className="flex h-svh overflow-hidden">
      <aside className="flex h-svh w-56 shrink-0 flex-col border-r border-border bg-card">
        <div className="flex h-14 shrink-0 items-center border-b border-border px-[var(--margin-desktop)]">
          <p className="font-headline-md text-foreground">Outreach</p>
        </div>
        <SidebarNav />
        <div className="mt-auto flex items-center justify-between gap-2 border-t border-border px-3 py-3">
          <p className="min-w-0 truncate font-label-md text-foreground">
            {workspace?.name ?? "Workspace"}
          </p>
          <LogoutButton />
        </div>
      </aside>
      <div className="flex min-h-0 min-w-0 flex-1 flex-col">{children}</div>
    </div>
  );
}

export default function AppLayout({ children }: { children: ReactNode }) {
  return (
    <Suspense fallback={<PageLoading />}>
      <AppShell>{children}</AppShell>
    </Suspense>
  );
}
