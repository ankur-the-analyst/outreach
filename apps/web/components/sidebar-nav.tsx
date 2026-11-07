"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { navItems } from "@/lib/nav";

export function SidebarNav() {
  const pathname = usePathname();
  return (
    <nav className="flex flex-1 flex-col gap-0.5 px-2 py-2">
      {navItems.map((item) => {
        const Icon = item.icon;
        const active =
          item.href === "/"
            ? pathname === "/"
            : pathname === item.href || pathname.startsWith(`${item.href}/`);
        return (
          <Link
            key={item.href}
            href={item.href}
            className={
              active
                ? "flex items-center gap-2 rounded-md bg-accent px-2 py-[7px] font-label-md text-primary"
                : "flex items-center gap-2 rounded-md px-2 py-[7px] font-label-md text-muted-foreground hover:bg-muted hover:text-foreground"
            }
          >
            <Icon className="size-[18px] shrink-0" strokeWidth={1.5} />
            {item.label}
          </Link>
        );
      })}
    </nav>
  );
}
