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

type State =
  | { status: "loading" }
  | { status: "loaded"; log: string }
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
 * Modal showing one build's console output, fetched when opened. A build still
 * running shows the log captured so far; "Refresh" re-fetches it, mirroring the
 * fork log dialog's read-only viewer.
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
    const controller = new AbortController();
    setState({ status: "loading" });

    getBuildLog(profile, service, job, number, controller.signal)
      .then((res) => {
        if (cancelled) return;
        setState(
          res.error
            ? { status: "error", error: res.error }
            : { status: "loaded", log: res.log },
        );
      })
      .catch((cause) => {
        if (cancelled || controller.signal.aborted) return;
        setState({
          status: "error",
          error: cause instanceof Error ? cause.message : String(cause),
        });
      });

    return () => {
      cancelled = true;
      controller.abort();
    };
  }, [profile, service, job, number, nonce]);

  // Jump to the end of the log once it loads, where the latest output is.
  useEffect(() => {
    if (state.status === "loaded" && logRef.current) {
      logRef.current.scrollTop = logRef.current.scrollHeight;
    }
  }, [state]);

  const refresh = useCallback(() => setNonce((n) => n + 1), []);

  return (
    <Dialog open={build !== null} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-[80vw]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <ScrollText className="size-4" aria-hidden />
            <span className="font-mono">{job}</span> #{build?.number}
          </DialogTitle>
          <DialogDescription>
            Console output —{" "}
            {build ? buildResultLabel(build.result, build.building) : ""}
          </DialogDescription>
        </DialogHeader>

        {state.status === "loading" && <Skeleton className="h-80 w-full" />}

        {state.status === "error" && (
          <Alert variant="destructive">
            <AlertCircle />
            <AlertDescription className="font-mono text-xs">
              {state.error}
            </AlertDescription>
          </Alert>
        )}

        {state.status === "loaded" && (
          <div
            ref={logRef}
            className="h-80 overflow-auto rounded-md border bg-muted/40 p-3 font-mono text-xs leading-relaxed"
            role="log"
          >
            {state.log.trim() === "" ? (
              <p className="text-muted-foreground">No console output.</p>
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
