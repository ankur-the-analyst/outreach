import { Button } from "@/components/ui/button";

export default function Home() {
  return (
    <main className="flex min-h-full flex-col items-start gap-4 p-[var(--margin-desktop)]">
      <h1 className="font-headline-xl text-foreground">Outreach</h1>
      <p className="font-body-md text-muted-foreground">Workspace shell.</p>
      <Button type="button">Continue</Button>
    </main>
  );
}
