import { useEffect, useRef } from "react";

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
import type { BackupSession } from "@/services/api";

import { STATUS_META } from "../status-meta";
import { useBackupSession } from "../hooks/useBackupSession";

interface BackupSessionDialogProps {
  /** Session to view; null keeps the dialog closed. */
  session: BackupSession | null;
  onOpenChange: (open: boolean) => void;
}

/**
 * Modal showing a single backup session's status and verbose log, read by id.
 * It attaches when opened — polling while the run is still in flight, or just
 * showing the stored log for a finished one — and offers "Cancel backup" while
 * running. Closing only stops polling; the backup (if any) keeps running.
 */
export function BackupSessionDialog({
  session,
  onOpenChange,
}: BackupSessionDialogProps) {
  const { state, attach, cancel, detach } = useBackupSession();
  const logRef = useRef<HTMLDivElement>(null);

  const id = session?.id;
  // Attach on open (keyed by id); stop polling — but keep the backup — on close.
  useEffect(() => {
    if (id) {
      attach(id);
    } else {
      detach();
    }
  }, [id, attach, detach]);

  // Keep the newest line in view as the log grows.
  useEffect(() => {
    const el = logRef.current;
    if (el) {
      el.scrollTop = el.scrollHeight;
    }
  }, [state.lines]);

  const running = state.status === "running";
  // Fall back to the row's status until the first poll lands.
  const status = state.status === "idle" ? session?.status : state.status;
  const meta = status ? STATUS_META[status] : undefined;
  const Icon = meta?.icon;

  return (
    <Dialog open={session !== null} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            {Icon && (
              <Icon
                className={cn(
                  "size-4",
                  meta?.spin && "animate-spin",
                  meta?.className,
                )}
                aria-hidden
              />
            )}
            {meta?.headline ?? "Backup"}
          </DialogTitle>
          <DialogDescription>
            Snapshot of <span className="font-mono">{session?.service}</span> in
            profile <span className="font-mono">{session?.profile}</span>.
          </DialogDescription>
        </DialogHeader>

        <div
          ref={logRef}
          className="h-72 overflow-auto rounded-md border bg-muted/40 p-3 font-mono text-xs leading-relaxed"
          role="log"
          aria-live="polite"
        >
          {state.lines.length === 0 ? (
            <p className="text-muted-foreground">
              {running ? "Loading…" : "No log output."}
            </p>
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
            <div className={cn("mt-2", meta?.className)}>
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
