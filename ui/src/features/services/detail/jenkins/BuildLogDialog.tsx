import { AlertCircle, RotateCw, ScrollText } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";

import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { getBuildLog } from "@/services/api";

import { buildResultLabel } from "./jobStatus";

/** How often to poll for new console output while a build is still running. */
const POLL_INTERVAL = 1500;

type State =
  | { status: "loading" }
  | { status: "active"; log: string; live: boolean }
  | { status: "error"; error: string };

interface BuildLogDialogProps {
  profile: string;
  service: string;
  job: string;
  /** Build number whose console output to show; null keeps the dialog closed. */
  build: { number: number; result?: string; building: boolean } | null;
  onOpenChange: (open: boolean) => void;
}

/**
 * Modal showing one build's console output, streamed via long polling. It reads
 * Jenkins's progressive log from the last byte offset every {@link POLL_INTERVAL}
 * and appends new output, stopping automatically once the build finishes
 * (`more` is false). "Refresh" restarts the stream from the top.
 */
export function BuildLogDialog({
  profile,
  service,
  job,
  build,
  onOpenChange,
}: BuildLogDialogProps) {
  const [state, setState] = useState<State>({ status: "loading" });
  const [nonce, setNonce] = useState(0);
  const logRef = useRef<HTMLDivElement>(null);

  const number = build?.number ?? null;

  useEffect(() => {
    if (number === null) return;

    let cancelled = false;
    let timer: ReturnType<typeof setTimeout> | null = null;
    const controller = new AbortController();
    let offset = 0;
    let acc = "";
    setState({ status: "loading" });

    const poll = async () => {
      try {
        const res = await getBuildLog(
          profile,
          service,
          job,
          number,
          offset,
          controller.signal,
        );
        if (cancelled) return;
        if (res.error) {
          setState({ status: "error", error: res.error });
          return;
        }
        if (res.text) acc += res.text;
        offset = res.offset;
        setState({ status: "active", log: acc, live: res.more });
        if (res.more) timer = setTimeout(() => void poll(), POLL_INTERVAL);
      } catch (cause) {
        if (cancelled || controller.signal.aborted) return;
        setState({
          status: "error",
          error: cause instanceof Error ? cause.message : String(cause),
        });
      }
    };
    void poll();

    return () => {
      cancelled = true;
      controller.abort();
      if (timer) clearTimeout(timer);
    };
  }, [profile, service, job, number, nonce]);

  // Follow the tail: keep the newest output in view as the log grows.
  useEffect(() => {
    if (state.status === "active" && logRef.current) {
      logRef.current.scrollTop = logRef.current.scrollHeight;
    }
  }, [state]);

  const refresh = useCallback(() => setNonce((n) => n + 1), []);

  const live = state.status === "active" && state.live;

  return (
    <Dialog open={build !== null} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-[80vw]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <ScrollText className="size-4" aria-hidden />
            <span className="font-mono">{job}</span> #{build?.number}
            {live && (
              <span className="ml-1 inline-flex items-center gap-1.5 text-xs font-medium text-sky-600 dark:text-sky-400">
                <span
                  className="size-2 rounded-full bg-sky-500 motion-safe:animate-pulse"
                  aria-hidden
                />
                Live
              </span>
            )}
          </DialogTitle>
          <DialogDescription>
            Console output —{" "}
            {build ? buildResultLabel(build.result, build.building) : ""}
          </DialogDescription>
        </DialogHeader>

        {state.status === "loading" && <Skeleton className="h-[80vh] w-full" />}

        {state.status === "error" && (
          <Alert variant="destructive">
            <AlertCircle />
            <AlertDescription className="font-mono text-xs">
              {state.error}
            </AlertDescription>
          </Alert>
        )}

        {state.status === "active" && (
          <div
            ref={logRef}
            className="h-[80vh] overflow-auto rounded-md border bg-muted/40 p-3 font-mono text-xs leading-relaxed"
            role="log"
            aria-live="polite"
          >
            {state.log.trim() === "" ? (
              <p className="text-muted-foreground">
                {state.live ? "Waiting for output…" : "No console output."}
              </p>
            ) : (
              <pre className="whitespace-pre-wrap break-all">{state.log}</pre>
            )}
          </div>
        )}

        <DialogFooter>
          <Button
            variant="outline"
            onClick={refresh}
            disabled={state.status === "loading"}
          >
            <RotateCw aria-hidden />
            Refresh
          </Button>
          <Button onClick={() => onOpenChange(false)}>Close</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
