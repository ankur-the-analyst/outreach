import Link from "next/link";
import { PageBody, PageHeader, Pill } from "@/components/page-header";
import { ReviewQueue } from "@/components/review-queue";
import type { Campaign, Draft, Mailbox } from "@/lib/api";
import {
  campaignStatusLabel,
  intentLabel,
  mailboxStatus,
  mailboxUsageLabel,
  reviewBucket,
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
          label="Queued to approve"
          value={pending.length}
          hint="Needs a human"
        />
        <Metric
          label="Can still undo"
          value={undo.length}
          hint="A few minutes left"
        />
        <Metric
          label="Sent (in review)"
          value={sent.length}
          hint="Already went out"
        />
        <Metric
          label="Paused mailboxes"
          value={paused.length}
          hint={paused[0]?.email ?? "None paused"}
          danger={paused.length > 0}
        />
      </div>
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-12">
        <section className="rounded-md border border-border bg-card lg:col-span-8">
          <div className="flex items-center justify-between border-b border-border px-4 py-3">
            <h2 className="font-headline-sm text-foreground">Needs you</h2>
            <Link
              href="/review"
              className="font-label-md text-primary hover:underline"
            >
              Full review
            </Link>
          </div>
          <div className="p-4">
            {needsYou.length === 0 ? (
              <p className="font-body-md text-muted-foreground">
                Nothing waiting.
              </p>
            ) : (
              <ReviewQueue drafts={needsYou} mailboxes={mailboxes} />
            )}
          </div>
        </section>
        <div className="flex flex-col gap-4 lg:col-span-4">
          <section className="rounded-md border border-border bg-card p-4">
            <div className="mb-3 flex items-center justify-between">
              <h2 className="font-headline-sm text-foreground">Mailboxes</h2>
              <Link
                href="/mailboxes"
                className="font-label-sm text-primary hover:underline"
              >
                Manage
              </Link>
            </div>
            {mailboxes.length === 0 ? (
              <p className="font-body-md text-muted-foreground">
                Connect a mailbox first.
              </p>
            ) : (
              <ul className="flex flex-col gap-2">
                {mailboxes.map((mb) => (
                  <li
                    key={mb.id}
                    className="rounded-md border border-border bg-muted/50 px-3 py-2"
                  >
                    <div className="flex items-center justify-between gap-2">
                      <p className="truncate font-label-md text-foreground">
                        {mb.email}
                      </p>
                      <Pill
                        tone={
                          mb.status === "paused" || mb.status === "disconnected"
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
                  </li>
                ))}
              </ul>
            )}
          </section>
          <section className="rounded-md border border-border bg-card p-4">
            <div className="mb-3 flex items-center justify-between">
              <h2 className="font-headline-sm text-foreground">Campaigns</h2>
              <Link
                href="/campaigns"
                className="font-label-sm text-primary hover:underline"
              >
                View all
              </Link>
            </div>
            {!snapshot ? (
              <p className="font-body-md text-muted-foreground">
                No campaigns yet.
              </p>
            ) : (
              <div className="rounded-md border border-border bg-muted/50 p-3">
                <div className="flex flex-wrap items-center gap-2">
                  <p className="font-headline-sm text-foreground">
                    {snapshot.name}
                  </p>
                  <Pill tone="secondary">
                    {intentLabel[snapshot.intent] ?? snapshot.intent}
                  </Pill>
                  <Pill
                    tone={snapshot.status === "running" ? "success" : "muted"}
                  >
                    {campaignStatusLabel[snapshot.status] ?? snapshot.status}
                  </Pill>
                </div>
                <p className="mt-2 font-body-sm text-muted-foreground">
                  {snapshot.steps.filter((s) => s.kind === "email").length}{" "}
                  emails
                </p>
              </div>
            )}
          </section>
        </div>
      </div>
      </div>
      </PageBody>
    </div>
  );
}

function Metric({
  label,
  value,
  hint,
  danger,
}: {
  label: string;
  value: number;
  hint: string;
  danger?: boolean;
}) {
  return (
    <div className="flex flex-col justify-between rounded-md border border-border bg-card p-4">
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
    </div>
  );
}
