import type { ReactNode } from "react";
import { Spinner } from "@/components/ui/spinner";

export function PageHeader({
  title,
  actions,
}: {
  title: string;
  actions?: ReactNode;
}) {
  return (
    <header className="flex min-h-14 shrink-0 flex-wrap items-center justify-between gap-3 border-b border-border px-[var(--margin-desktop)] py-2.5">
      <h1 className="font-headline-md text-foreground">{title}</h1>
      {actions ? (
        <div className="flex flex-wrap items-center justify-end gap-2">
          {actions}
        </div>
      ) : null}
    </header>
  );
}

export function Panel({
  title,
  description,
  actions,
  children,
}: {
  title: string;
  description?: string;
  actions?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section className="flex flex-col gap-3 rounded-md border border-border bg-card p-4">
      <div className="flex min-h-8 flex-wrap items-start justify-between gap-2">
        <div className="min-w-0">
          <h2 className="font-headline-sm text-foreground">{title}</h2>
          {description ? (
            <p className="mt-1 font-body-sm text-muted-foreground">
              {description}
            </p>
          ) : null}
        </div>
        {actions ? (
          <div className="flex flex-wrap items-center gap-2">{actions}</div>
        ) : null}
      </div>
      {children}
    </section>
  );
}

export function EmptyState({
  title,
  description,
  action,
  compact,
}: {
  title: string;
  description?: string;
  action?: ReactNode;
  compact?: boolean;
}) {
  return (
    <div
      className={
        compact
          ? "flex flex-col items-start gap-2 rounded-md border border-dashed border-border bg-muted/40 px-4 py-6"
          : "flex flex-col items-start gap-3 rounded-md border border-dashed border-border bg-card px-5 py-10"
      }
    >
      <div className="flex flex-col gap-1">
        <p className="font-headline-sm text-foreground">{title}</p>
        {description ? (
          <p className="max-w-prose font-body-md text-muted-foreground">
            {description}
          </p>
        ) : null}
      </div>
      {action ? <div className="flex flex-wrap gap-2">{action}</div> : null}
    </div>
  );
}

export function PageBody({ children }: { children: ReactNode }) {
  return (
    <div className="min-h-0 flex-1 overflow-y-auto p-[var(--margin-desktop)]">
      {children}
    </div>
  );
}

export function PageLoading() {
  return (
    <div className="flex min-h-svh w-full flex-1 items-center justify-center bg-background">
      <Spinner className="size-12 text-muted-foreground" />
    </div>
  );
}

export function Pill({
  children,
  tone = "muted",
}: {
  children: ReactNode;
  tone?: "muted" | "primary" | "secondary" | "success" | "danger";
}) {
  const tones = {
    muted: "bg-muted text-muted-foreground",
    primary: "bg-accent text-primary",
    secondary: "bg-secondary/15 text-secondary",
    success: "bg-success/15 text-success",
    danger: "bg-destructive/10 text-destructive",
  };
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 font-label-sm ${tones[tone]}`}
    >
      {children}
    </span>
  );
}
