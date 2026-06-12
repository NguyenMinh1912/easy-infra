import { useCallback, useEffect, useRef, useState } from "react";

import {
  cancelBackup,
  getBackup,
  startServiceBackup,
  type BackupStatus,
} from "@/services/api";

/** How often to poll a running backup for new log lines, in milliseconds. */
const POLL_INTERVAL = 800;

/** Reactive state of a single service's backup run. */
export interface BackupState {
  status: BackupStatus | "idle";
  /** Verbose log lines accumulated from polling. */
  lines: string[];
  /** Error message when status is "error". */
  error?: string;
  /** Snapshot folder name when the backup succeeded. */
  snapshot?: string;
}

const initial: BackupState = { status: "idle", lines: [] };

/**
 * Drive a backup of one service by polling. `start` kicks off (or re-attaches
 * to) the server-side session and polls it for new log lines until it settles;
 * `cancel` asks the server to stop it; `reset` stops polling and returns to idle
 * without cancelling — so closing the dialog leaves the backup running. Polling
 * is also torn down on unmount.
 */
export function useBackup(serviceName: string) {
  const [state, setState] = useState<BackupState>(initial);

  const sessionId = useRef<string | null>(null);
  const lastSeq = useRef(0);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const stopped = useRef(false);

  const stop = useCallback(() => {
    stopped.current = true;
    if (timer.current) {
      clearTimeout(timer.current);
      timer.current = null;
    }
  }, []);

  const poll = useCallback(async () => {
    const id = sessionId.current;
    if (stopped.current || !id) return;

    let res;
    try {
      res = await getBackup(id, lastSeq.current);
    } catch (cause) {
      if (!stopped.current) {
        setState((s) => ({ ...s, status: "error", error: String(cause) }));
      }
      return;
    }
    if (stopped.current) return;

    if (res.logs.length > 0) {
      lastSeq.current = res.logs[res.logs.length - 1].seq;
    }
    setState((s) => ({
      ...s,
      status: res.session.status,
      lines:
        res.logs.length > 0
          ? [...s.lines, ...res.logs.map((l) => l.line)]
          : s.lines,
      error: res.session.error ?? s.error,
      snapshot: res.session.snapshot ?? s.snapshot,
    }));

    if (res.session.status === "running") {
      timer.current = setTimeout(() => void poll(), POLL_INTERVAL);
    }
  }, []);

  const start = useCallback(async () => {
    stop();
    stopped.current = false;
    sessionId.current = null;
    lastSeq.current = 0;
    setState({ status: "running", lines: [] });

    try {
      const session = await startServiceBackup(serviceName);
      if (stopped.current) return;
      sessionId.current = session.id;
      void poll();
    } catch (cause) {
      if (!stopped.current) {
        setState((s) => ({ ...s, status: "error", error: String(cause) }));
      }
    }
  }, [serviceName, stop, poll]);

  const cancel = useCallback(async () => {
    const id = sessionId.current;
    if (!id) return;
    try {
      await cancelBackup(id);
    } catch {
      // The next poll surfaces the outcome; ignore a failed cancel request.
    }
  }, []);

  const reset = useCallback(() => {
    stop();
    sessionId.current = null;
    lastSeq.current = 0;
    setState(initial);
  }, [stop]);

  useEffect(() => () => stop(), [stop]);

  return { state, start, cancel, reset };
}
