import { useState } from "react";
import {
  Activity,
  Eraser,
  GitFork,
  Play,
  Save,
  type LucideIcon,
} from "lucide-react";
import { toast } from "sonner";

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import type { ServiceDefinition } from "@/types/service";

import { ApplyLogDialog } from "./ApplyLogDialog";
import { BackupLogDialog } from "./BackupLogDialog";
import { ForkDialog } from "./ForkDialog";
import { ForkLogDialog } from "./ForkLogDialog";
import { SnapshotSelectDialog } from "./SnapshotSelectDialog";

interface ServiceActionsProps {
  service: ServiceDefinition;
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
  { id: "status", label: "Service status", icon: Activity },
  { id: "backup", label: "Backup", icon: Save },
  { id: "apply", label: "Apply", icon: Play },
  { id: "fork", label: "Fork to local", icon: GitFork },
  { id: "clean", label: "Clean", icon: Eraser, destructive: true },
];

/**
 * The navbar action bar for a single service: status, backup, apply, fork, and
 * clean, laid out as a horizontal row of buttons. Backup, apply, and fork are
 * wired to the API — backup confirms then streams the snapshot's verbose log
 * into a modal; apply picks a backup version then streams the restore's log;
 * fork picks a seed (an existing version or a fresh backup), launches a local
 * container with the same config, and streams the fork's log. The remaining
 * operations run server-side but are not exposed yet, so they announce that they
 * are coming rather than calling a missing endpoint.
 */
export function ServiceActions({ service }: ServiceActionsProps) {
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [logOpen, setLogOpen] = useState(false);
  const [snapshotOpen, setSnapshotOpen] = useState(false);
  const [applyOpen, setApplyOpen] = useState(false);
  const [applySnapshot, setApplySnapshot] = useState("");
  const [forkOpen, setForkOpen] = useState(false);
  const [forkLogOpen, setForkLogOpen] = useState(false);
  const [forkSnapshot, setForkSnapshot] = useState("");

  const run = (action: ServiceAction) => {
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

      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Back up {service.name}?</AlertDialogTitle>
            <AlertDialogDescription>
              This snapshots the active profile's{" "}
              <span className="font-mono">{service.name}</span> data into a new
              backup folder. You can follow the progress in the next step.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                setConfirmOpen(false);
                setLogOpen(true);
              }}
            >
              Back up
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <BackupLogDialog
        serviceName={service.name}
        open={logOpen}
        onOpenChange={setLogOpen}
      />

      <SnapshotSelectDialog
        serviceName={service.name}
        open={snapshotOpen}
        onOpenChange={setSnapshotOpen}
        onApply={(snapshot) => {
          setApplySnapshot(snapshot);
          setApplyOpen(true);
        }}
      />

      <ApplyLogDialog
        serviceName={service.name}
        snapshot={applySnapshot}
        open={applyOpen}
        onOpenChange={setApplyOpen}
      />

      <ForkDialog
        serviceName={service.name}
        open={forkOpen}
        onOpenChange={setForkOpen}
        onFork={(snapshot) => {
          setForkSnapshot(snapshot);
          setForkLogOpen(true);
        }}
      />

      <ForkLogDialog
        serviceName={service.name}
        snapshot={forkSnapshot}
        open={forkLogOpen}
        onOpenChange={setForkLogOpen}
      />
    </>
  );
}
