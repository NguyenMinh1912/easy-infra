import { useCallback, useEffect, useRef, useState } from "react";

import {
  cancelBackup,
  getBackup,
  type BackupSession,
  type BackupStatus,
} from "@/services/api";

/** How often to poll a running session for new log lines, in milliseconds. */
const POLL_INTERVAL = 800;

/** Reactive state of a single backup or apply run. */
export interface SessionState {
  status: BackupStatus | "idle";
  /** Verbose log lines accumulated from polling. */
  lines: string[];
  /** Error message when status is "error". */
  error?: string;
  /** Snapshot folder name the run backed up to or restored from. */
  snapshot?: string;
}

const initial: SessionState = { status: "idle", lines: [] };

/**
 * Drive a backup or apply by polling. The kind of run is decided by the
 * injected `starter`, which kicks off (or re-attaches to) the server-side
 * session; the hook then polls it for new log lines until it settles. `cancel`
 * asks the server to stop it; `reset` stops polling and returns to idle without
 * cancelling — so closing the dialog leaves the run going. Polling is also torn
 * down on unmount.
 *
 * Pass a memoized `starter` (e.g. via `useCallback`) so `start` keeps a stable
 * identity across renders.
 */
export function useBackupSession(starter: () => Promise<BackupSession>) {
  const [state, setState] = useState<SessionState>(initial);

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
      const session = await starter();
      if (stopped.current) return;
      sessionId.current = session.id;
      void poll();
    } catch (cause) {
      if (!stopped.current) {
        setState((s) => ({ ...s, status: "error", error: String(cause) }));
      }
    }
  }, [starter, stop, poll]);

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
