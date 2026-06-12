import { useCallback, useEffect, useRef, useState } from "react";

import { streamServiceBackup, type BackupResult } from "@/services/api";

/** Where a backup run is in its lifecycle. */
export type BackupStatus =
  | "idle"
  | "running"
  | "success"
  | "unsupported"
  | "error";

/** Reactive state of a single service's backup run. */
export interface BackupState {
  status: BackupStatus;
  /** Verbose log lines accumulated from the stream. */
  lines: string[];
  /** Error message when status is "error". */
  error?: string;
  /** Snapshot folder name when the backup succeeded. */
  snapshot?: string;
}

const initial: BackupState = { status: "idle", lines: [] };

/**
 * Drive a streaming backup of one service. `start` kicks off the SSE stream and
 * accumulates log lines as they arrive; `cancel` aborts an in-flight run; and
 * `reset` returns to idle (e.g. when the modal closes). Any in-flight stream is
 * aborted on unmount so a closed dialog never updates state.
 */
export function useBackup(serviceName: string) {
  const [state, setState] = useState<BackupState>(initial);
  const controllerRef = useRef<AbortController | null>(null);

  const start = useCallback(() => {
    controllerRef.current?.abort();
    const controller = new AbortController();
    controllerRef.current = controller;
    setState({ status: "running", lines: [] });

    void streamServiceBackup(
      serviceName,
      {
        onLog: (line) =>
          setState((s) => ({ ...s, lines: [...s.lines, line] })),
        onDone: (result: BackupResult) =>
          setState((s) => ({
            ...s,
            status: result.status === "unsupported" ? "unsupported" : "success",
            snapshot: result.snapshot,
          })),
        onError: (message) =>
          setState((s) => ({ ...s, status: "error", error: message })),
      },
      controller.signal,
    );
  }, [serviceName]);

  const cancel = useCallback(() => {
    controllerRef.current?.abort();
  }, []);

  const reset = useCallback(() => {
    controllerRef.current?.abort();
    controllerRef.current = null;
    setState(initial);
  }, []);

  useEffect(() => () => controllerRef.current?.abort(), []);

  return { state, start, cancel, reset };
}
