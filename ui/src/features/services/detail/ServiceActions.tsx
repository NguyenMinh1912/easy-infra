import { useState } from "react";
import {
  Activity,
  Eraser,
  GitFork,
  Play,
  Save,
  Settings,
  type LucideIcon,
} from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import type { ServiceInstance } from "@/types/service";

import { ApplyLogDialog } from "./ApplyLogDialog";
import { BackupLogDialog } from "./BackupLogDialog";
import { BackupSelectDialog } from "./BackupSelectDialog";
import { ForkDialog } from "./ForkDialog";
import { ForkLogDialog } from "./ForkLogDialog";
import { ServiceSettingsDialog } from "./ServiceSettingsDialog";
import { SnapshotSelectDialog } from "./SnapshotSelectDialog";

interface ServiceActionsProps {
  service: ServiceInstance;
  /**
   * Profile the service is viewed under. Scopes every action — settings,
   * backup, apply, fork — to that profile rather than whatever profile is
   * active server-side, and is required to open and save the service's
   * settings.
   */
  profile: string;
  /** Called after settings are saved, so the detail screen can reload. */
  onChanged?: () => void;
}

/** One operation offered in the service action menu. */
interface ServiceAction {
  id: string;
  label: string;
  icon: LucideIcon;
  /** Styled and confirmed differently — `clean` discards data. */
  destructive?: boolean;
}

const ACTIONS: ServiceAction[] = [
  { id: "settings", label: "Settings", icon: Settings },
  { id: "status", label: "Service status", icon: Activity },
  { id: "backup", label: "Backup", icon: Save },
  { id: "apply", label: "Apply", icon: Play },
  { id: "fork", label: "Fork to local", icon: GitFork },
  { id: "clean", label: "Clean", icon: Eraser, destructive: true },
];

/**
 * The navbar action bar for a single service: settings, status, backup, apply,
 * fork, and clean, laid out as a horizontal row of buttons. Settings opens a
 * modal to edit this service's config for the profile. Backup, apply, and fork
 * are wired to the API — backup confirms then streams the snapshot's verbose log
 * into a modal; apply picks a backup version then streams the restore's log;
 * fork picks a seed (an existing version or a fresh backup), launches a local
 * container with the same config, and streams the fork's log. The remaining
 * operations run server-side but are not exposed yet, so they announce that they
 * are coming rather than calling a missing endpoint.
 */
export function ServiceActions({ service, profile, onChanged }: ServiceActionsProps) {
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [logOpen, setLogOpen] = useState(false);
  const [backupBuckets, setBackupBuckets] = useState<string[] | undefined>(
    undefined,
  );
  const [snapshotOpen, setSnapshotOpen] = useState(false);
  const [applyOpen, setApplyOpen] = useState(false);
  const [applySnapshot, setApplySnapshot] = useState("");
  const [forkOpen, setForkOpen] = useState(false);
  const [forkLogOpen, setForkLogOpen] = useState(false);
  const [forkSnapshot, setForkSnapshot] = useState("");
  const [forkPort, setForkPort] = useState<number | undefined>(undefined);

  const run = (action: ServiceAction) => {
    if (action.id === "settings") {
      setSettingsOpen(true);
      return;
    }
    if (action.id === "backup") {
      setConfirmOpen(true);
      return;
    }
    if (action.id === "apply") {
      setSnapshotOpen(true);
      return;
    }
    if (action.id === "fork") {
      setForkOpen(true);
      return;
    }
    toast.info(`"${action.label}" is not available yet`, {
      description: `Running ${action.id} for ${service.name} from the UI is coming soon.`,
    });
  };

  return (
    <>
      <div
        className="flex flex-wrap items-center gap-2"
        role="toolbar"
        aria-label="Service actions"
      >
        {ACTIONS.map((action) => {
          const Icon = action.icon;
          return (
            <Button
              key={action.id}
              variant={action.destructive ? "destructive" : "outline"}
              size="sm"
              onClick={() => run(action)}
            >
              <Icon aria-hidden />
              {action.label}
            </Button>
          );
        })}
      </div>

      {settingsOpen && (
        <ServiceSettingsDialog
          service={service}
          profile={profile}
          onClose={() => setSettingsOpen(false)}
          onSaved={onChanged}
        />
      )}

      <BackupSelectDialog
        serviceName={service.id}
        profile={profile}
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        onBackup={(buckets) => {
          setBackupBuckets(buckets);
          setLogOpen(true);
        }}
      />

      <BackupLogDialog
        serviceName={service.id}
        profile={profile}
        buckets={backupBuckets}
        open={logOpen}
        onOpenChange={setLogOpen}
      />

      <SnapshotSelectDialog
        serviceName={service.id}
        profile={profile}
        open={snapshotOpen}
        onOpenChange={setSnapshotOpen}
        onApply={(snapshot) => {
          setApplySnapshot(snapshot);
          setApplyOpen(true);
        }}
      />

      <ApplyLogDialog
        serviceName={service.id}
        profile={profile}
        snapshot={applySnapshot}
        open={applyOpen}
        onOpenChange={setApplyOpen}
      />

      <ForkDialog
        serviceName={service.id}
        profile={profile}
        open={forkOpen}
        onOpenChange={setForkOpen}
        onFork={(snapshot, port) => {
          setForkSnapshot(snapshot);
          setForkPort(port);
          setForkLogOpen(true);
        }}
      />

      <ForkLogDialog
        serviceName={service.id}
        profile={profile}
        snapshot={forkSnapshot}
        port={forkPort}
        open={forkLogOpen}
        onOpenChange={setForkLogOpen}
      />
    </>
  );
}
