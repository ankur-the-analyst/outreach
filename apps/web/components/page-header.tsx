import type { ReactNode } from "react";

export function PageHeader({
  title,
  actions,
}: {
  title: string;
  actions?: ReactNode;
}) {
  return (
    <header className="flex h-14 shrink-0 items-center justify-between gap-3 border-b border-border px-[var(--margin-desktop)]">
      <h1 className="font-headline-md text-foreground">{title}</h1>
      {actions ? (
        <div className="flex flex-wrap items-center justify-end gap-2">
          {actions}
        </div>
      ) : null}
    </header>
  );
}

export function PageBody({ children }: { children: ReactNode }) {
  return (
    <div className="min-h-0 flex-1 overflow-y-auto p-[var(--margin-desktop)]">
      {children}
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
