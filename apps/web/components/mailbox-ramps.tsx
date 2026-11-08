import { warmupKindLabel } from "@/lib/labels";
import type { WarmupProfile } from "@/lib/api";

export function MailboxRamps({ profiles }: { profiles: WarmupProfile[] }) {
  return (
    <div className="flex flex-col gap-3 rounded-md border border-border bg-card p-4">
      <h2 className="font-headline-sm text-foreground">Send ramps</h2>
      {profiles.length === 0 ? (
        <p className="font-body-md text-muted-foreground">No ramps yet.</p>
      ) : (
        <ul className="flex flex-col gap-2">
          {profiles.map((p) => (
            <li key={p.id} className="flex flex-wrap items-baseline justify-between gap-2">
              <p className="font-label-md text-foreground">
                {p.name || warmupKindLabel(p.kind)}
              </p>
              <p className="font-body-sm text-muted-foreground">
                {p.first_day != null
                  ? `Starts at ${p.first_day}/day`
                  : warmupKindLabel(p.kind)}
              </p>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
