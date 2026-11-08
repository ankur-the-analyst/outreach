import { healthMetricLabel } from "@/lib/labels";
import type { HealthSample } from "@/lib/api";

function latestByMetric(samples: HealthSample[]): HealthSample[] {
  const seen = new Set<string>();
  const out: HealthSample[] = [];
  for (const s of samples) {
    if (seen.has(s.metric)) {
      continue;
    }
    seen.add(s.metric);
    out.push(s);
  }
  return out;
}

function formatValue(sample: HealthSample): string {
  if (sample.metric === "sync_lag_s") {
    const n = Math.round(sample.value);
    if (n < 60) {
      return `${n}s ago`;
    }
    const mins = Math.round(n / 60);
    return `${mins} min ago`;
  }
  return String(Math.round(sample.value));
}

export function MailboxHealth({ samples }: { samples: HealthSample[] }) {
  const latest = latestByMetric(samples);
  return (
    <div className="flex flex-col gap-3 rounded-md border border-border bg-card p-4">
      <h2 className="font-headline-sm text-foreground">Mailbox health</h2>
      {latest.length === 0 ? (
        <p className="font-body-md text-muted-foreground">No readings yet.</p>
      ) : (
        <ul className="flex flex-col gap-2">
          {latest.map((s) => (
            <li
              key={s.metric}
              className="flex flex-wrap items-baseline justify-between gap-2"
            >
              <p className="font-label-md text-foreground">
                {healthMetricLabel(s.metric)}
              </p>
              <p className="font-body-sm text-muted-foreground">
                {formatValue(s)}
              </p>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
