import Link from "next/link";
import { EmptyState, PageBody, PageHeader, Panel, Pill } from "@/components/page-header";
import type { Campaign, Draft, Mailbox } from "@/lib/api";
import {
  campaignStatusLabel,
  intentLabel,
  mailboxStatus,
  mailboxUsageLabel,
  reviewBucket,
  sendStatus,
} from "@/lib/labels";

export function OverviewBoard({
  drafts,
  mailboxes,
  campaigns,
  usageByMailbox,
}: {
  drafts: Draft[];
  mailboxes: Mailbox[];
  campaigns: Campaign[];
  usageByMailbox: Record<string, number>;
}) {
  const pending = drafts.filter((d) => reviewBucket(d) === "pending");
  const undo = drafts.filter((d) => reviewBucket(d) === "undo");
  const sent = drafts.filter((d) => reviewBucket(d) === "sent");
  const paused = mailboxes.filter((m) => m.status === "paused");
  const needsYou = [...undo, ...pending].slice(0, 5);
  const snapshot = campaigns.find((c) => c.status === "running") ?? campaigns[0];

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <PageHeader title="Overview" />
      <PageBody>
        <div className="flex flex-col gap-6">
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <Metric
              href="/review"
              label="Queued to approve"
              value={pending.length}
              hint="Needs a human"
            />
            <Metric
              href="/review"
              label="Can still undo"
              value={undo.length}
              hint="A few minutes left"
            />
            <Metric
              href="/review"
              label="Sent (in review)"
              value={sent.length}
              hint="Already went out"
            />
            <Metric
              href="/mailboxes"
              label="Paused mailboxes"
              value={paused.length}
              hint={paused[0]?.email ?? "None paused"}
              danger={paused.length > 0}
            />
          </div>
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-12">
            <div className="lg:col-span-8">
              <Panel
                title="Needs you"
                actions={
                  <Link
                    href="/review"
                    className="font-label-md text-primary hover:underline"
                  >
                    Review
                  </Link>
                }
              >
                {needsYou.length === 0 ? (
                  <EmptyState
                    compact
                    title="Nothing waiting"
                    description="New drafts to approve or undo will show up here."
                    action={
                      <Link
                        href="/review"
                        className="font-label-md text-primary hover:underline"
                      >
                        Open review
                      </Link>
                    }
                  />
                ) : (
                  <ul className="flex flex-col gap-2">
                    {needsYou.map((d) => {
                      const mailbox = mailboxes.find((m) => m.id === d.mailbox_id);
                      return (
                        <li key={d.id}>
                          <Link
                            href="/review"
                            className="flex flex-wrap items-center justify-between gap-2 rounded-md border border-border bg-muted/50 px-3 py-2 transition-colors hover:bg-muted focus-visible:ring-3 focus-visible:ring-ring/50"
                          >
                            <div className="min-w-0">
                              <p className="truncate font-label-md text-foreground">
                                {d.subject || "Untitled"}
                              </p>
                              <p className="mt-0.5 truncate font-body-sm text-muted-foreground">
                                To {d.to}
                                {mailbox ? ` · via ${mailbox.email}` : ""}
                              </p>
                            </div>
                            <Pill
                              tone={
                                reviewBucket(d, mailbox) === "undo"
                                  ? "primary"
                                  : "muted"
                              }
                            >
                              {sendStatus(d, mailbox)}
                            </Pill>
                          </Link>
                        </li>
                      );
                    })}
                  </ul>
                )}
              </Panel>
            </div>
            <div className="flex flex-col gap-4 lg:col-span-4">
              <Panel
                title="Mailboxes"
                actions={
                  <Link
                    href="/mailboxes"
                    className="font-label-sm text-primary hover:underline"
                  >
                    Manage
                  </Link>
                }
              >
                {mailboxes.length === 0 ? (
                  <EmptyState
                    compact
                    title="No mailboxes"
                    description="Connect one before a campaign can send."
                    action={
                      <Link
                        href="/mailboxes"
                        className="font-label-md text-primary hover:underline"
                      >
                        Connect mailbox
                      </Link>
                    }
                  />
                ) : (
                  <ul className="flex flex-col gap-2">
                    {mailboxes.map((mb) => (
                      <li key={mb.id}>
                        <Link
                          href="/mailboxes"
                          className="block rounded-md border border-border bg-muted/50 px-3 py-2 transition-colors hover:bg-muted focus-visible:ring-3 focus-visible:ring-ring/50"
                        >
                        <div className="flex items-center justify-between gap-2">
                          <p className="truncate font-label-md text-foreground">
                            {mb.email}
                          </p>
                          <Pill
                            tone={
                              mb.status === "paused" ||
                              mb.status === "disconnected"
                                ? "danger"
                                : "success"
                            }
                          >
                            {mailboxStatus(mb.status)}
                          </Pill>
                        </div>
                        <p className="mt-1 font-body-sm tabular-nums text-muted-foreground">
                          {mailboxUsageLabel(
                            usageByMailbox[mb.id] ?? 0,
                            mb.effective_daily_limit,
                          )}
                        </p>
                        </Link>
                      </li>
                    ))}
                  </ul>
                )}
              </Panel>
              <Panel
                title="Campaigns"
                actions={
                  <Link
                    href="/campaigns"
                    className="font-label-sm text-primary hover:underline"
                  >
                    View all
                  </Link>
                }
              >
                {!snapshot ? (
                  <EmptyState
                    compact
                    title="No campaigns yet"
                    description="Create one after a mailbox is connected."
                    action={
                      <Link
                        href="/campaigns"
                        className="font-label-md text-primary hover:underline"
                      >
                        Campaigns
                      </Link>
                    }
                  />
                ) : (
                  <Link
                    href="/campaigns"
                    className="block rounded-md border border-border bg-muted/50 p-3 transition-colors hover:bg-muted focus-visible:ring-3 focus-visible:ring-ring/50"
                  >
                    <div className="flex flex-wrap items-center gap-2">
                      <p className="font-headline-sm text-foreground">
                        {snapshot.name}
                      </p>
                      <Pill tone="secondary">
                        {intentLabel[snapshot.intent] ?? snapshot.intent}
                      </Pill>
                      <Pill
                        tone={
                          snapshot.status === "running" ? "success" : "muted"
                        }
                      >
                        {campaignStatusLabel[snapshot.status] ?? snapshot.status}
                      </Pill>
                    </div>
                    <p className="mt-2 font-body-sm text-muted-foreground">
                      {snapshot.steps.filter((s) => s.kind === "email").length}{" "}
                      emails
                    </p>
                  </Link>
                )}
              </Panel>
            </div>
          </div>
        </div>
      </PageBody>
    </div>
  );
}

function Metric({
  href,
  label,
  value,
  hint,
  danger,
}: {
  href: string;
  label: string;
  value: number;
  hint: string;
  danger?: boolean;
}) {
  return (
    <Link
      href={href}
      className="flex flex-col justify-between rounded-md border border-border bg-card p-4 transition-colors hover:bg-muted/40 focus-visible:ring-3 focus-visible:ring-ring/50"
    >
      <span className="font-label-md uppercase tracking-wider text-muted-foreground">
        {label}
      </span>
      <span
        className={`font-numeric-metric my-2 ${danger ? "text-destructive" : "text-foreground"}`}
      >
        {value}
      </span>
      <span className="truncate border-t border-border pt-2 font-body-sm text-muted-foreground">
        {hint}
      </span>
    </Link>
  );
}
