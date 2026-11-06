"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { navItems } from "@/lib/nav";

export function SidebarNav() {
  const pathname = usePathname();
  return (
    <nav className="flex flex-1 flex-col gap-0.5">
      {navItems.map((item) => {
        const active = pathname === item.href;
        return (
          <Link
            key={item.href}
            href={item.href}
            className={
              active
                ? "rounded-md bg-accent px-2.5 py-1.5 font-label-md text-accent-foreground"
                : "rounded-md px-2.5 py-1.5 font-label-md text-muted-foreground hover:bg-muted hover:text-foreground"
            }
          >
            {item.label}
          </Link>
        );
      })}
    </nav>
  );
}
