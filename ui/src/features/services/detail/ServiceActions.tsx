import { useState } from "react";
import { Activity, Eraser, Play, Save, type LucideIcon } from "lucide-react";
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

import { BackupLogDialog } from "./BackupLogDialog";

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
  { id: "clean", label: "Clean", icon: Eraser, destructive: true },
];

/**
 * The navbar action bar for a single service: status, backup, apply, and clean,
 * laid out as a horizontal row of buttons. Backup is wired to the API — it
 * confirms, then streams the snapshot's verbose log into a modal. The remaining
 * operations run server-side but are not exposed yet, so they announce that they
 * are coming rather than calling a missing endpoint.
 */
export function ServiceActions({ service }: ServiceActionsProps) {
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [logOpen, setLogOpen] = useState(false);

  const run = (action: ServiceAction) => {
    if (action.id === "backup") {
      setConfirmOpen(true);
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
    </>
  );
}
