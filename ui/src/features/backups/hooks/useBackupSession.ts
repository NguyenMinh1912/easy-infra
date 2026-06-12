import { useCallback, useEffect, useRef, useState } from "react";

import { cancelBackup, getBackup, type BackupStatus } from "@/services/api";

/** How often to poll a running session for new log lines, in milliseconds. */
const POLL_INTERVAL = 800;

/** Reactive state of a backup session being viewed by id. */
export interface BackupSessionState {
  status: BackupStatus | "idle";
  /** Verbose log lines accumulated from polling. */
  lines: string[];
  /** Error message when status is "error". */
  error?: string;
  /** Snapshot folder name when the backup succeeded. */
  snapshot?: string;
}

const initial: BackupSessionState = { status: "idle", lines: [] };

/**
 * View an existing backup session by id: fetch its logs and, while it is still
 * running, keep polling for new lines until it settles. `attach` (re)starts
 * watching a session from the beginning; `cancel` asks the server to stop a
 * running one; `detach` stops polling and returns to idle. Polling is also torn
 * down on unmount.
 *
 * Unlike the service-detail backup hook, this never *starts* a backup — it only
 * observes one that already exists, so it works for finished runs too.
 */
export function useBackupSession() {
  const [state, setState] = useState<BackupSessionState>(initial);

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

  const attach = useCallback(
    (id: string) => {
      stop();
      stopped.current = false;
      sessionId.current = id;
      lastSeq.current = 0;
      setState({ status: "running", lines: [] });
      void poll();
    },
    [stop, poll],
  );

  const cancel = useCallback(async () => {
    const id = sessionId.current;
    if (!id) return;
    try {
      await cancelBackup(id);
    } catch {
      // The next poll surfaces the outcome; ignore a failed cancel request.
    }
  }, []);

  const detach = useCallback(() => {
    stop();
    sessionId.current = null;
    lastSeq.current = 0;
    setState(initial);
  }, [stop]);

  useEffect(() => () => stop(), [stop]);

  return { state, attach, cancel, detach };
}
