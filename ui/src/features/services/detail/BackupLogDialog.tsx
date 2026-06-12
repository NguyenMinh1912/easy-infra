import { useEffect, useRef } from "react";
import {
  CheckCircle2,
  Info,
  Loader2,
  XCircle,
  type LucideIcon,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { cn } from "@/lib/utils";

import { useBackup, type BackupStatus } from "./useBackup";

interface BackupLogDialogProps {
  /** Service to back up; also drives the stream once the dialog opens. */
  serviceName: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

/** Per-status header copy and icon for the running backup. */
const STATUS_META: Record<
  BackupStatus,
  { icon: LucideIcon; label: string; spin?: boolean; className?: string }
> = {
  idle: { icon: Loader2, label: "" },
  running: { icon: Loader2, label: "Backing up…", spin: true },
  success: {
    icon: CheckCircle2,
    label: "Backup complete",
    className: "text-emerald-600 dark:text-emerald-400",
  },
  unsupported: {
    icon: Info,
    label: "Backup not supported yet",
    className: "text-muted-foreground",
  },
  error: {
    icon: XCircle,
    label: "Backup failed",
    className: "text-destructive",
  },
};

/**
 * Modal showing a live, terminal-style verbose log while a service is backed
 * up. It starts the stream when opened and accumulates log lines via
 * {@link useBackup}; the footer offers Cancel while running and Close once the
 * run settles. Closing the dialog aborts any in-flight backup.
 */
export function BackupLogDialog({
  serviceName,
  open,
  onOpenChange,
}: BackupLogDialogProps) {
  const { state, start, reset } = useBackup(serviceName);
  const logRef = useRef<HTMLDivElement>(null);

  // Start the stream on open and reset back to idle on close.
  useEffect(() => {
    if (open) {
      start();
    } else {
      reset();
    }
  }, [open, start, reset]);

  // Keep the newest line in view as the log grows.
  useEffect(() => {
    const el = logRef.current;
    if (el) {
      el.scrollTop = el.scrollHeight;
    }
  }, [state.lines]);

  const running = state.status === "running";
  const meta = STATUS_META[state.status];
  const Icon = meta.icon;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Icon
              className={cn("size-4", meta.spin && "animate-spin", meta.className)}
              aria-hidden
            />
            {meta.label || "Backup"}
          </DialogTitle>
          <DialogDescription>
            Snapshotting <span className="font-mono">{serviceName}</span> for the
            active profile.
          </DialogDescription>
        </DialogHeader>

        <div
          ref={logRef}
          className="h-72 overflow-auto rounded-md border bg-muted/40 p-3 font-mono text-xs leading-relaxed"
          role="log"
          aria-live="polite"
        >
          {state.lines.length === 0 && running ? (
            <p className="text-muted-foreground">Starting backup…</p>
          ) : (
            state.lines.map((line, i) => (
              <div key={i} className="whitespace-pre-wrap break-all">
                {line}
              </div>
            ))
          )}
          {state.status === "error" && state.error && (
            <div className="mt-2 whitespace-pre-wrap break-all text-destructive">
              {state.error}
            </div>
          )}
          {state.status === "success" && state.snapshot && (
            <div className={cn("mt-2", meta.className)}>
              Saved snapshot {state.snapshot}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button
            variant={running ? "outline" : "default"}
            onClick={() => onOpenChange(false)}
          >
            {running ? "Cancel" : "Close"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
