export const navItems = [
  { href: "/", label: "Overview" },
  { href: "/inbox", label: "Inbox" },
  { href: "/tracker", label: "Tracker" },
  { href: "/contacts", label: "Contacts" },
  { href: "/campaigns", label: "Campaigns" },
  { href: "/review", label: "Review" },
  { href: "/evidence", label: "Evidence" },
  { href: "/jobs", label: "Jobs" },
  { href: "/mailboxes", label: "Mailboxes" },
  { href: "/profiles", label: "Profiles" },
  { href: "/settings", label: "Settings" },
] as const;

export function navLabel(href: string) {
  return navItems.find((item) => item.href === href)?.label ?? "Overview";
}
