import { useCallback, useEffect, useRef } from "react";
import {
  Ban,
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
import { startServiceBackup } from "@/services/api";
import { cn } from "@/lib/utils";

import { useBackupSession, type SessionState } from "./useBackupSession";

interface BackupLogDialogProps {
  /** Service to back up; also drives the stream once the dialog opens. */
  serviceName: string;
  /**
   * Buckets to include (minio); omitted backs up everything. Chosen in the
   * preceding {@link BackupSelectDialog}.
   */
  buckets?: string[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

/** Per-status header copy and icon. */
const STATUS_META: Record<
  SessionState["status"],
  { icon: LucideIcon; label: string; spin?: boolean; className?: string }
> = {
  idle: { icon: Loader2, label: "Backup" },
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
  cancelled: {
    icon: Ban,
    label: "Backup cancelled",
    className: "text-muted-foreground",
  },
  error: {
    icon: XCircle,
    label: "Backup failed",
    className: "text-destructive",
  },
};

/**
 * Modal showing a backup's verbose log, polled from the server. It starts (or
 * re-attaches to) the run when opened and accumulates log lines via
 * {@link useBackupSession}. While running it offers "Cancel backup" (stops it
 * server-side) and "Close" — closing only stops polling, so the backup keeps
 * running and can be re-attached by reopening.
 */
export function BackupLogDialog({
  serviceName,
  buckets,
  open,
  onOpenChange,
}: BackupLogDialogProps) {
  const starter = useCallback(
    () => startServiceBackup(serviceName, buckets),
    [serviceName, buckets],
  );
  const { state, start, cancel, reset } = useBackupSession(starter);
  const logRef = useRef<HTMLDivElement>(null);

  // Start (or re-attach) on open; stop polling — but keep the backup — on close.
  useEffect(() => {
    if (open) {
      void start();
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
            {meta.label}
          </DialogTitle>
          <DialogDescription>
            Snapshotting <span className="font-mono">{serviceName}</span> for the
            active profile. This runs in the background — closing keeps it going.
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
          {running && (
            <Button variant="destructive" onClick={() => void cancel()}>
              Cancel backup
            </Button>
          )}
          <Button
            variant={running ? "outline" : "default"}
            onClick={() => onOpenChange(false)}
          >
            Close
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
