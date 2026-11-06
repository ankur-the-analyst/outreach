import type { ReactNode } from "react";
import { Suspense } from "react";
import { redirect } from "next/navigation";

export const instant = false;
import { LogoutButton } from "@/components/logout-button";
import { SidebarNav } from "@/components/sidebar-nav";
import { getMe, getWorkspace } from "@/lib/api";

async function AppShell({ children }: { children: ReactNode }) {
  const me = await getMe();
  if (!me) {
    redirect("/login");
  }
  const workspace = await getWorkspace();

  return (
    <div className="flex min-h-full">
      <aside className="flex w-56 shrink-0 flex-col gap-6 border-r border-border bg-card p-4">
        <p className="font-headline-md text-foreground">Outreach</p>
        <SidebarNav />
        <div className="mt-auto flex flex-col gap-2">
          <p className="truncate font-label-md text-muted-foreground">
            {workspace?.name ?? "Workspace"}
          </p>
          <LogoutButton />
        </div>
      </aside>
      <div className="min-w-0 flex-1 p-[var(--margin-desktop)]">{children}</div>
    </div>
  );
}

export default function AppLayout({ children }: { children: ReactNode }) {
  return (
    <Suspense
      fallback={
        <div className="min-h-full bg-background p-[var(--margin-desktop)] font-body-md text-muted-foreground">
          Loading…
        </div>
      }
    >
      <AppShell>{children}</AppShell>
    </Suspense>
  );
}
