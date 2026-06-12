import { useCallback, useEffect, useRef } from "react";
import {
  Ban,
  CheckCircle2,
  GitFork,
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
import { notifyProfilesChanged } from "@/features/profiles";
import { startServiceFork } from "@/services/api";
import { cn } from "@/lib/utils";

import { useBackupSession, type SessionState } from "./useBackupSession";

interface ForkLogDialogProps {
  /** Service to fork; also drives the stream once the dialog opens. */
  serviceName: string;
  /** Snapshot version to seed from; an empty string takes a fresh backup. */
  snapshot: string;
  /** Local port to publish the fork on; undefined keeps the source's port. */
  port?: number;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

/** Per-status header copy and icon. */
const STATUS_META: Record<
  SessionState["status"],
  { icon: LucideIcon; label: string; spin?: boolean; className?: string }
> = {
  idle: { icon: GitFork, label: "Fork to local" },
  running: { icon: Loader2, label: "Forking…", spin: true },
  success: {
    icon: CheckCircle2,
    label: "Fork complete",
    className: "text-emerald-600 dark:text-emerald-400",
  },
  unsupported: {
    icon: Info,
    label: "Fork not supported yet",
    className: "text-muted-foreground",
  },
  cancelled: {
    icon: Ban,
    label: "Fork cancelled",
    className: "text-muted-foreground",
  },
  error: {
    icon: XCircle,
    label: "Fork failed",
    className: "text-destructive",
  },
};

/**
 * Modal showing a fork's verbose log, polled from the server. It starts (or
 * re-attaches to) the run when opened and accumulates log lines via
 * {@link useBackupSession}. While running it offers "Cancel fork" and "Close" —
 * closing only stops polling, so the fork keeps running. On success it nudges
 * the sidebar to reload so the new `local` profile shows up.
 */
export function ForkLogDialog({
  serviceName,
  snapshot,
  port,
  open,
  onOpenChange,
}: ForkLogDialogProps) {
  const starter = useCallback(
    () => startServiceFork(serviceName, snapshot, port),
    [serviceName, snapshot, port],
  );
  const { state, start, cancel, reset } = useBackupSession(starter);
  const logRef = useRef<HTMLDivElement>(null);

  // Start (or re-attach) on open; stop polling — but keep the fork — on close.
  useEffect(() => {
    if (open) {
      void start();
    } else {
      reset();
    }
  }, [open, start, reset]);

  // The local profile only exists once a fork succeeds; refresh the sidebar then.
  useEffect(() => {
    if (state.status === "success") {
      notifyProfilesChanged();
    }
  }, [state.status]);

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
  const seed = snapshot || "a fresh backup";

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
            Forking <span className="font-mono">{serviceName}</span> to a local
            container, seeded from{" "}
            <span className="font-mono">{seed}</span>. This runs in the
            background — closing keeps it going.
          </DialogDescription>
        </DialogHeader>

        <div
          ref={logRef}
          className="h-72 overflow-auto rounded-md border bg-muted/40 p-3 font-mono text-xs leading-relaxed"
          role="log"
          aria-live="polite"
        >
          {state.lines.length === 0 && running ? (
            <p className="text-muted-foreground">Starting fork…</p>
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
              Forked to the <span className="font-mono">local</span> profile.
            </div>
          )}
        </div>

        <DialogFooter>
          {running && (
            <Button variant="destructive" onClick={() => void cancel()}>
              Cancel fork
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
