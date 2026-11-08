import { Suspense } from "react";
import { PageLoading } from "@/components/page-header";
import { SettingsBoard } from "@/components/settings-board";
import { getMembers, getSenderIdentity, getWorkspace } from "@/lib/api";

export const instant = false;

async function SettingsBody() {
  const [workspace, identity, members] = await Promise.all([
    getWorkspace(),
    getSenderIdentity(),
    getMembers(),
  ]);
  return (
    <main className="flex min-h-0 flex-1 flex-col">
      <SettingsBoard
        workspace={workspace}
        identity={identity}
        members={members?.members ?? []}
      />
    </main>
  );
}

export default function SettingsPage() {
  return (
    <Suspense fallback={<PageLoading />}>
      <SettingsBody />
    </Suspense>
  );
}
