import type { LucideIcon } from "lucide-react";
import {
  Briefcase,
  CircleUser,
  FileCheck,
  Inbox,
  LayoutDashboard,
  LineChart,
  Mail,
  Megaphone,
  Settings,
  ShieldCheck,
  Users,
} from "lucide-react";

export type NavItem = {
  href: string;
  label: string;
  icon: LucideIcon;
};

export const navGroups: { label: string; items: NavItem[] }[] = [
  {
    label: "Today",
    items: [
      { href: "/", label: "Overview", icon: LayoutDashboard },
      { href: "/inbox", label: "Inbox", icon: Inbox },
      { href: "/tracker", label: "Tracker", icon: LineChart },
      { href: "/review", label: "Review", icon: ShieldCheck },
    ],
  },
  {
    label: "Outreach",
    items: [
      { href: "/contacts", label: "Contacts", icon: Users },
      { href: "/campaigns", label: "Campaigns", icon: Megaphone },
      { href: "/jobs", label: "Jobs", icon: Briefcase },
      { href: "/evidence", label: "Evidence", icon: FileCheck },
      { href: "/profiles", label: "Profiles", icon: CircleUser },
    ],
  },
  {
    label: "Workspace",
    items: [
      { href: "/mailboxes", label: "Mailboxes", icon: Mail },
      { href: "/settings", label: "Settings", icon: Settings },
    ],
  },
];

export const navItems: NavItem[] = navGroups.flatMap((group) => group.items);

export function navLabel(href: string) {
  return navItems.find((item) => item.href === href)?.label ?? "Overview";
}
