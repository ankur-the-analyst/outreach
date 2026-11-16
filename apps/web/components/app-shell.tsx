"use client";

import { useEffect, useState, type ReactNode } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { PanelLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { LogoutButton } from "@/components/logout-button";
import { SidebarNav } from "@/components/sidebar-nav";

const storageKey = "outreach-sidebar";

export function AppShell({
  name,
  children,
}: {
  name: string;
  children: ReactNode;
}) {
  const pathname = usePathname();
  const [collapsed, setCollapsed] = useState(false);
  const [mobileOpen, setMobileOpen] = useState(false);

  useEffect(() => {
    setCollapsed(window.localStorage.getItem(storageKey) === "collapsed");
  }, []);

  useEffect(() => {
    setMobileOpen(false);
  }, [pathname]);

  function toggleCollapsed() {
    setCollapsed((cur) => {
      const next = !cur;
      window.localStorage.setItem(storageKey, next ? "collapsed" : "open");
      return next;
    });
  }

  return (
    <div className="flex h-dvh overflow-hidden">
      <aside
        className={
          collapsed
            ? "hidden h-dvh w-14 shrink-0 flex-col border-r border-border bg-card md:flex"
            : "hidden h-dvh w-56 shrink-0 flex-col border-r border-border bg-card md:flex"
        }
      >
        <SidebarChrome
          name={name}
          collapsed={collapsed}
          onToggle={toggleCollapsed}
        />
      </aside>
      <div className="flex min-h-0 min-w-0 flex-1 flex-col bg-background">
        <div className="flex h-14 shrink-0 items-center gap-2 border-b border-border bg-card px-3 md:hidden">
          <Button
            type="button"
            variant="ghost"
            size="icon"
            aria-label="Open menu"
            onClick={() => setMobileOpen(true)}
          >
            <PanelLeft className="size-[18px]" strokeWidth={1.5} />
          </Button>
          <p className="font-headline-md text-foreground">Outreach</p>
        </div>
        {children}
      </div>
      <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
        <SheetContent side="left" className="w-72 max-w-[85vw] p-0 sm:max-w-xs">
          <SheetHeader className="border-b border-border px-4 py-3">
            <SheetTitle className="font-headline-md">Outreach</SheetTitle>
          </SheetHeader>
          <div className="flex min-h-0 flex-1 flex-col">
            <SidebarNav />
            <div className="mt-auto flex items-center justify-between gap-2 border-t border-border px-3 py-3">
              <p className="min-w-0 truncate font-label-md text-foreground">{name}</p>
              <LogoutButton />
            </div>
          </div>
        </SheetContent>
      </Sheet>
    </div>
  );
}

function SidebarChrome({
  name,
  collapsed,
  onToggle,
}: {
  name: string;
  collapsed: boolean;
  onToggle: () => void;
}) {
  return (
    <>
      <div
        className={
          collapsed
            ? "flex h-14 shrink-0 items-center justify-center border-b border-border"
            : "flex h-14 shrink-0 items-center justify-between gap-2 border-b border-border px-3"
        }
      >
        {collapsed ? null : (
          <Link href="/" className="truncate font-headline-md text-foreground">
            Outreach
          </Link>
        )}
        <Button
          type="button"
          variant="ghost"
          size="icon"
          aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
          aria-expanded={!collapsed}
          onClick={onToggle}
        >
          <PanelLeft className="size-[18px]" strokeWidth={1.5} />
        </Button>
      </div>
      <SidebarNav collapsed={collapsed} />
      <div
        className={
          collapsed
            ? "mt-auto flex items-center justify-center border-t border-border py-3"
            : "mt-auto flex items-center justify-between gap-2 border-t border-border px-3 py-3"
        }
      >
        {collapsed ? null : (
          <p className="min-w-0 truncate font-label-md text-foreground">{name}</p>
        )}
        <LogoutButton />
      </div>
    </>
  );
}
