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
import { startServiceApply } from "@/services/api";
import { cn } from "@/lib/utils";

import { useBackupSession, type SessionState } from "./useBackupSession";

interface ApplyLogDialogProps {
  /** Service to apply; also drives the stream once the dialog opens. */
  serviceName: string;
  /** Profile the service is viewed under; the restore lands in it. */
  profile?: string;
  /**
   * Profile the snapshot is read from. Equals `profile` for a same-profile
   * restore, or another profile when restoring its backup of the same service.
   */
  sourceProfile?: string;
  /** Snapshot version to restore; an empty string applies the latest. */
  snapshot: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

/** Per-status header copy and icon. */
const STATUS_META: Record<
  SessionState["status"],
  { icon: LucideIcon; label: string; spin?: boolean; className?: string }
> = {
  idle: { icon: Loader2, label: "Apply" },
  running: { icon: Loader2, label: "Applying…", spin: true },
  success: {
    icon: CheckCircle2,
    label: "Apply complete",
    className: "text-emerald-600 dark:text-emerald-400",
  },
  unsupported: {
    icon: Info,
    label: "Apply not supported yet",
    className: "text-muted-foreground",
  },
  cancelled: {
    icon: Ban,
    label: "Apply cancelled",
    className: "text-muted-foreground",
  },
  error: {
    icon: XCircle,
    label: "Apply failed",
    className: "text-destructive",
  },
};

/**
 * Modal showing an apply's verbose log, polled from the server. It starts (or
 * re-attaches to) the run when opened and accumulates log lines via
 * {@link useBackupSession}. While running it offers "Cancel apply" (stops it
 * server-side) and "Close" — closing only stops polling, so the apply keeps
 * running and can be re-attached by reopening.
 */
export function ApplyLogDialog({
  serviceName,
  profile,
  sourceProfile,
  snapshot,
  open,
  onOpenChange,
}: ApplyLogDialogProps) {
  const starter = useCallback(
    () => startServiceApply(serviceName, snapshot, profile, sourceProfile),
    [serviceName, snapshot, profile, sourceProfile],
  );
  const { state, start, cancel, reset } = useBackupSession(starter);
  const logRef = useRef<HTMLDivElement>(null);

  // Start (or re-attach) on open; stop polling — but keep the apply — on close.
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
  const version = snapshot || "the latest snapshot";
  // Surface the source profile only when it differs from the viewed one, so a
  // same-profile restore reads exactly as before.
  const fromProfile =
    sourceProfile && sourceProfile !== profile ? sourceProfile : undefined;

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
            Restoring <span className="font-mono">{serviceName}</span> for the
            active profile from{" "}
            <span className="font-mono">{version}</span>
            {fromProfile && (
              <>
                {" "}
                in profile <span className="font-mono">{fromProfile}</span>
              </>
            )}
            . This runs in the background — closing keeps it going.
          </DialogDescription>
        </DialogHeader>

        <div
          ref={logRef}
          className="h-72 overflow-auto rounded-md border bg-muted/40 p-3 font-mono text-xs leading-relaxed"
          role="log"
          aria-live="polite"
        >
          {state.lines.length === 0 && running ? (
            <p className="text-muted-foreground">Starting apply…</p>
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
          {state.status === "success" && (
            <div className={cn("mt-2", meta.className)}>
              Restored from {state.snapshot || version}
            </div>
          )}
        </div>

        <DialogFooter>
          {running && (
            <Button variant="destructive" onClick={() => void cancel()}>
              Cancel apply
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
