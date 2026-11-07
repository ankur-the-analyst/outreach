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

export const navItems: {
  href: string;
  label: string;
  icon: LucideIcon;
}[] = [
  { href: "/", label: "Overview", icon: LayoutDashboard },
  { href: "/inbox", label: "Inbox", icon: Inbox },
  { href: "/tracker", label: "Tracker", icon: LineChart },
  { href: "/contacts", label: "Contacts", icon: Users },
  { href: "/campaigns", label: "Campaigns", icon: Megaphone },
  { href: "/review", label: "Review", icon: ShieldCheck },
  { href: "/evidence", label: "Evidence", icon: FileCheck },
  { href: "/jobs", label: "Jobs", icon: Briefcase },
  { href: "/mailboxes", label: "Mailboxes", icon: Mail },
  { href: "/profiles", label: "Profiles", icon: CircleUser },
  { href: "/settings", label: "Settings", icon: Settings },
];

export function navLabel(href: string) {
  return navItems.find((item) => item.href === href)?.label ?? "Overview";
}
