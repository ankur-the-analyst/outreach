import { Suspense } from "react";
import { PageLoading } from "@/components/page-header";
import { SettingsBoard } from "@/components/settings-board";
import {
  getMembers,
  getModelSettings,
  getSenderIdentity,
  getWorkspace,
} from "@/lib/api";

export const instant = false;

async function SettingsBody() {
  const [workspace, identity, members, models] = await Promise.all([
    getWorkspace(),
    getSenderIdentity(),
    getMembers(),
    getModelSettings(),
  ]);
  return (
    <main className="flex min-h-0 flex-1 flex-col">
      <SettingsBoard
        workspace={workspace}
        identity={identity}
        members={members?.members ?? []}
        models={models}
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
