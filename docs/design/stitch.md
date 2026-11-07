# Stitch frames (Now)

Source: Stitch HTML for Overview, Review, Campaigns, Tracker (Nov 2026). **Layout and density only.** Colors, type, radius stay `theme.css`. Do not copy the HTML `tailwind.config` (it drifted). Do not ship Tailwind CDN or Material Symbols; use shadcn + Lucide.

## Token remap (ignore Stitch hex)

| Stitch used | Use instead |
|---|---|
| `#a03b13` / `#c1522a` primary | `#D9643A` white label |
| `#ffdbd0` / `#ffb59c` wash | `#FDEEE9` |
| `#f9f9fe` / `#f3f3f9` canvas | `#F4F4F6` |
| `#ededf3` wells | `#F0F0F3` |
| `#1a1c20` text | `#1E2024` |
| `#57423b` / `#8b726a` meta (warm brown) | `#6B7280` |
| `#dec0b7` border | `#E2E2E8` |
| `#00685f` / `#2b8178` teal | `#006A61` / `#36AFA1` |
| Radius `0.5rem` / `0.75rem` cards | `0.25rem` (pills stay full) |

Campaigns frame already hard-codes the locked hex in the body; other frames do not. Always CSS variables.

## Shell (all app screens)

- Sidebar 224px, white, 1px right border. Wordmark Outreach (Noto Serif).
- Nav with Lucide, 11 items as today. Active: `#FDEEE9` wash + `#D9643A` label.
- Footer: workspace name, Sign out. Drop “Active Account” and the fake online dot unless we have real presence.
- **Drop the top chrome bar** (“Workspace Mode”, “High Intent”, duplicate avatar). Status belongs on Overview/Review, not every page.
- Main: gray canvas, 24px pad. No second global header.

## Keep from Stitch

**Overview:** 4 metric tiles (queued to approve, can still undo, sent last 24h, paused mailboxes). “Needs you” list with undo/approve. Empty: “Nothing waiting.” Mailbox cards with daily used/cap. One campaign snapshot.

**Review:** Filter pills (Needs approval / You can still undo / Checking if it sent / Sent). Draft cards: subject, To, mailbox, status, body `68ch`, evidence chips, gate-failure well, Approve / Undo send / Reject / one Regenerate. Compose in a modal, not a stacked form on the same page.

**Campaigns:** Header + Create (primary) + Add person (outline). Metric tiles only if we have real counts (else omit, do not fake 37.5%). Campaign cards: name, type pill, status pill, unsubscribe, email count, mailbox, Pause/Start/Stop, add person. Sequence as a vertical stepper. Create in a right drawer (name, type, mailbox, new/day, strategy, steps). Add person in a dialog (campaign, contact, why you’re writing). Sender identity stays a real form, not only a callout.

**Tracker:** Table + optional inspector. Columns: contact, company, campaign, stage, outcome (guess vs confirmed), last activity. Search + campaign + stage filters. Row opens inspector: stage, campaign, email, latest thread (`68ch`), next step, Open in Inbox, Stop. Pagination when we have it.

## Do not ship (Stitch invented)

- Enroll / enrolled / Campaign Architect / Direct Assignment / Confirm Assignment
- Response rate, “Active Leads”, match score 94%, “High Intent Match”, “Desk Live”
- Export CSV, Columns picker (Later)
- Diagnostic logs, Force polling, relay node, Attempt 1/3, “300 seconds”
- Configure gate checks, Filter policies, Preview empty state as a product control
- Fake SPF/DKIM on Mailpit, “Preserving deliverability” marketing
- Business-days-only wait (Now is wait N days)
- Header “checking if it sent” on every screen

## Copy

Operator language only. Add person / in this campaign. Campaign type. You can still undo. Checking if it sent.

## Build order

Shell first (sidebar + drop top bar), then restyle pages we already have (Review, Campaigns, Contacts, Evidence, Mailboxes, Inbox), then Overview and Tracker when those APIs exist (M6.4). Do not block M6.3 scheduler on this.
