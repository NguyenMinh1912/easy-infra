import {
  AlertCircle,
  CheckCircle2,
  ChevronRight,
  Eraser,
  Play,
  Terminal,
} from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button, buttonVariants } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import { useAsync } from "@/hooks/useAsync";
import { useRemainingHeight } from "@/hooks/useRemainingHeight";
import { executeQuery, getDatabases } from "@/services/api";
import type { QueryResult } from "@/types/console";

import { DbSelector } from "./DbSelector";

interface RedisConsoleProps {
  /** Profile whose saved connection config the command runs against. */
  profile: string;
  /** Service name within the profile (the API path segment). */
  service: string;
  /** Logical database commands run against, shared with the Keys tab. */
  db: number;
  onDbChange: (db: number) => void;
  /** Open a returned key in the Keys tab (jumps tabs and selects it). */
  onOpenKey: (key: string) => void;
}

/** Outcome of one logged command execution. */
type EntryState =
  | { status: "running" }
  | { status: "done"; result: QueryResult }
  | { status: "failed"; error: string };

/** One command in the result log, newest first. */
interface LogEntry {
  id: number;
  command: string;
  /** Database the command ran against. */
  db: number;
  /** Wall-clock time the command was issued. */
  at: Date;
  state: EntryState;
}

/** Cap the result log so a long session can't grow without bound. */
const MAX_LOG = 50;

/**
 * Commands that destroy data or can stall the server: running one prompts for
 * confirmation first. KEYS is included for its server-blocking scan, the rest
 * for irreversible writes.
 */
const GUARDED: Record<string, { title: string; description: string }> = {
  FLUSHDB: {
    title: "Flush this database?",
    description:
      "FLUSHDB permanently deletes every key in the selected database. This cannot be undone.",
  },
  FLUSHALL: {
    title: "Flush all databases?",
    description:
      "FLUSHALL permanently deletes every key in every database. This cannot be undone.",
  },
  DEL: {
    title: "Delete these keys?",
    description: "DEL permanently removes the named keys. This cannot be undone.",
  },
  UNLINK: {
    title: "Delete these keys?",
    description:
      "UNLINK permanently removes the named keys (asynchronously). This cannot be undone.",
  },
  KEYS: {
    title: "Run KEYS?",
    description:
      "KEYS scans the entire keyspace in one blocking pass and can freeze a busy server. Prefer SCAN for large databases.",
  },
};

/**
 * redis-cli-style console against one profile's Redis: type a command, run it,
 * and read the reply. Results stack into a log (newest first) rather than
 * replacing, so earlier output stays visible. SCAN replies render with a
 * labelled cursor and a clickable key list; destructive commands prompt for
 * confirmation; command failures come back inside the response envelope and
 * render with the offending command echoed.
 */
export function RedisConsole({
  profile,
  service,
  db,
  onDbChange,
  onOpenKey,
}: RedisConsoleProps) {
  const [command, setCommand] = useState("");
  const [log, setLog] = useState<LogEntry[]>([]);
  const [running, setRunning] = useState(false);
  // A guarded command awaiting confirmation before it runs.
  const [pending, setPending] = useState<{ command: string; verb: string } | null>(
    null,
  );

  // The database count drives the selector; if introspection fails, fall back
  // to Redis's default of 16 so the picker still works.
  const dbState = useAsync(
    (signal) => getDatabases(profile, service, signal),
    [profile, service],
  );
  const dbCount =
    dbState.status === "success" && dbState.data.count > 0
      ? dbState.data.count
      : 16;

  const idRef = useRef(0);
  const controllerRef = useRef<AbortController | null>(null);
  useEffect(() => () => controllerRef.current?.abort(), []);

  const runCommand = useCallback(
    (raw: string) => {
      const cmd = raw.trim();
      if (!cmd) return;
      controllerRef.current?.abort();
      const controller = new AbortController();
      controllerRef.current = controller;

      const id = ++idRef.current;
      const entry: LogEntry = {
        id,
        command: cmd,
        db,
        at: new Date(),
        state: { status: "running" },
      };
      setLog((prev) => [entry, ...prev].slice(0, MAX_LOG));
      setRunning(true);

      const settle = (state: EntryState) =>
        setLog((prev) =>
          prev.map((e) => (e.id === id ? { ...e, state } : e)),
        );

      executeQuery(profile, service, cmd, controller.signal, db)
        .then((result) => {
          if (controller.signal.aborted) return;
          settle(
            result.error
              ? { status: "failed", error: result.error }
              : { status: "done", result },
          );
        })
        .catch((cause: unknown) => {
          if (controller.signal.aborted) return;
          settle({
            status: "failed",
            error: cause instanceof Error ? cause.message : String(cause),
          });
        })
        .finally(() => {
          if (!controller.signal.aborted) setRunning(false);
        });
    },
    [profile, service, db],
  );

  // Run the command, but first divert guarded verbs to a confirmation prompt.
  const submit = useCallback(
    (raw: string) => {
      const cmd = raw.trim();
      if (!cmd) return;
      const verb = cmd.split(/\s+/)[0]?.toUpperCase() ?? "";
      if (GUARDED[verb]) {
        setPending({ command: cmd, verb });
        return;
      }
      runCommand(cmd);
    },
    [runCommand],
  );

  // Fill the height left below the input so the log uses the whole viewport and
  // scrolls in place instead of stopping short (matching the key browser).
  const { ref: logRef, maxHeight } = useRemainingHeight<HTMLDivElement>();

  const guard = pending ? GUARDED[pending.verb] : null;

  return (
    <div className="space-y-4">
      <form
        className="flex items-center gap-2"
        onSubmit={(e) => {
          e.preventDefault();
          submit(command);
        }}
      >
        <DbSelector db={db} count={dbCount} onChange={onDbChange} />
        <Input
          value={command}
          onChange={(e) => setCommand(e.target.value)}
          placeholder="Command, e.g. SCAN 0 MATCH user:*"
          aria-label="Redis command"
          className="min-w-0 flex-1 font-mono"
          autoComplete="off"
          spellCheck={false}
        />
        <Button type="submit" size="sm" disabled={running || command.trim() === ""}>
          <Play aria-hidden /> Run
        </Button>
        {log.length > 0 && (
          <Button
            type="button"
            size="sm"
            variant="ghost"
            onClick={() => setLog([])}
            aria-label="Clear result log"
          >
            <Eraser aria-hidden /> Clear
          </Button>
        )}
      </form>

      <div
        ref={logRef}
        style={{ maxHeight: maxHeight ?? undefined }}
        className="min-h-[12rem] space-y-3 overflow-auto"
      >
        {log.length === 0 ? (
          <div className="flex h-full min-h-[12rem] flex-col items-center justify-center gap-3 rounded-md border border-dashed border-border p-10 text-center">
            <span className="flex size-10 items-center justify-center rounded-lg bg-muted">
              <Terminal className="size-5 text-muted-foreground" aria-hidden />
            </span>
            <p className="text-sm text-muted-foreground">
              Run a command to see its reply here. Results stack newest-first.
            </p>
          </div>
        ) : (
          log.map((entry) => (
            <LogEntryCard
              key={entry.id}
              entry={entry}
              onRecall={setCommand}
              onOpenKey={onOpenKey}
              onScanNext={(next) => {
                setCommand(next);
                if (!running) runCommand(next);
              }}
            />
          ))
        )}
      </div>

      <AlertDialog
        open={pending !== null}
        onOpenChange={(open) => {
          if (!open) setPending(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{guard?.title}</AlertDialogTitle>
            <AlertDialogDescription>
              {guard?.description}
              <span className="mt-2 block font-mono text-xs text-foreground">
                {pending?.command}
              </span>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              className={cn(
                pending?.verb !== "KEYS" &&
                  buttonVariants({ variant: "destructive" }),
              )}
              onClick={() => {
                if (pending) runCommand(pending.command);
                setPending(null);
              }}
            >
              Run command
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

/** One result-log card: a header, the rendered reply, and a metadata line. */
function LogEntryCard({
  entry,
  onRecall,
  onOpenKey,
  onScanNext,
}: {
  entry: LogEntry;
  onRecall: (command: string) => void;
  onOpenKey: (key: string) => void;
  onScanNext: (next: string) => void;
}) {
  return (
    <div className="rounded-md border border-border">
      <div className="flex items-center gap-2 border-b border-border bg-muted/30 px-3 py-1.5">
        <StatusDot state={entry.state} />
        <button
          type="button"
          onClick={() => onRecall(entry.command)}
          title="Copy to the command input"
          className="min-w-0 flex-1 truncate text-left font-mono text-sm hover:underline"
        >
          {entry.command}
        </button>
        <span className="shrink-0 rounded border border-border px-1.5 text-[0.65rem] uppercase text-muted-foreground">
          db {entry.db}
        </span>
        <span className="shrink-0 text-xs text-muted-foreground">
          {entry.at.toLocaleTimeString()}
        </span>
      </div>
      <div className="p-3">
        <EntryBody entry={entry} onOpenKey={onOpenKey} onScanNext={onScanNext} />
      </div>
    </div>
  );
}

/** Coloured status indicator for a log entry. */
function StatusDot({ state }: { state: EntryState }) {
  if (state.status === "running") {
    return (
      <span
        className="size-2 shrink-0 animate-pulse rounded-full bg-muted-foreground"
        aria-label="Running"
      />
    );
  }
  if (state.status === "failed") {
    return (
      <span
        className="size-2 shrink-0 rounded-full bg-destructive"
        aria-label="Failed"
      />
    );
  }
  return (
    <span
      className="size-2 shrink-0 rounded-full bg-success"
      aria-label="Succeeded"
    />
  );
}

/** Renders an entry's body by status: skeleton, error, SCAN view, or reply. */
function EntryBody({
  entry,
  onOpenKey,
  onScanNext,
}: {
  entry: LogEntry;
  onOpenKey: (key: string) => void;
  onScanNext: (next: string) => void;
}) {
  if (entry.state.status === "running") {
    return <Skeleton className="h-8 w-2/3" />;
  }
  if (entry.state.status === "failed") {
    return (
      <Alert variant="destructive">
        <AlertCircle />
        <div>
          <AlertTitle>Command failed</AlertTitle>
          <AlertDescription className="font-mono text-xs">
            {entry.state.error}
          </AlertDescription>
        </div>
      </Alert>
    );
  }

  const result = entry.state.result;
  const scan = parseScan(result);
  if (scan) {
    return (
      <ScanResult
        command={entry.command}
        scan={scan}
        durationMs={result.durationMs}
        onOpenKey={onOpenKey}
        onScanNext={onScanNext}
      />
    );
  }
  return <ReplyResult result={result} />;
}

/** A parsed SCAN reply: the continuation cursor and the keys it returned. */
interface ScanReply {
  cursor: string;
  keys: string[];
}

/**
 * Recognise a SCAN reply and pull out its cursor and key list. The backend
 * keeps the reply structured as [cursor, [keys…]], so a match is a two-row
 * result whose second row holds an array.
 */
function parseScan(result: QueryResult): ScanReply | null {
  if (result.command !== "SCAN") return null;
  if (result.rows.length !== 2) return null;
  const keysCell = result.rows[1]?.[0];
  if (!Array.isArray(keysCell)) return null;
  return {
    cursor: String(result.rows[0]?.[0] ?? "0"),
    keys: keysCell.map((k) => String(k)),
  };
}

/**
 * SCAN reply: an explicit cursor label, a clickable key list, a "Scan next"
 * helper while the scan continues, and a clarified metadata line.
 */
function ScanResult({
  command,
  scan,
  durationMs,
  onOpenKey,
  onScanNext,
}: {
  command: string;
  scan: ScanReply;
  durationMs: number;
  onOpenKey: (key: string) => void;
  onScanNext: (next: string) => void;
}) {
  const complete = scan.cursor === "0";
  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-2 text-sm">
        <span className="font-medium">Cursor:</span>
        <span className="font-mono">{scan.cursor}</span>
        <span className="text-xs text-muted-foreground">
          {complete ? "(iteration complete)" : "(more to scan)"}
        </span>
        {!complete && (
          <Button
            type="button"
            size="sm"
            variant="secondary"
            className="ml-auto h-7 gap-1 px-2 text-xs"
            onClick={() => onScanNext(nextScanCommand(command, scan.cursor))}
          >
            <ChevronRight aria-hidden /> Scan next
          </Button>
        )}
      </div>

      <div className="space-y-1.5">
        <p className="text-xs font-medium text-muted-foreground">
          Keys ({scan.keys.length})
        </p>
        {scan.keys.length === 0 ? (
          <p className="text-sm text-muted-foreground">No keys on this page.</p>
        ) : (
          <ul className="flex flex-wrap gap-1.5">
            {scan.keys.map((key, i) => (
              <li key={`${key}-${i}`}>
                <button
                  type="button"
                  onClick={() => onOpenKey(key)}
                  title="Open in the Keys tab"
                  className="break-all rounded border border-border px-2 py-0.5 text-left font-mono text-xs hover:bg-muted hover:text-foreground"
                >
                  {key}
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>

      <p className="text-xs text-muted-foreground">
        SCAN · {scan.keys.length} {scan.keys.length === 1 ? "key" : "keys"} ·
        cursor {scan.cursor} · {durationMs} ms
      </p>
    </div>
  );
}

/**
 * A non-SCAN reply: a scalar shown as a value block, an array shown as an
 * indexed list, with a clarified metadata line.
 */
function ReplyResult({ result }: { result: QueryResult }) {
  const values = result.rows.map((row) => formatCell(row[0]));
  const single = values.length === 1;

  return (
    <div className="space-y-2">
      {result.rows.length === 0 ? (
        <div className="flex items-center gap-2 text-sm">
          <CheckCircle2 className="size-4 text-success" aria-hidden />
          <span className="font-mono">{result.command}</span>
        </div>
      ) : single ? (
        <ValueBlock value={values[0]} />
      ) : (
        <ol className="max-h-72 divide-y divide-border overflow-auto rounded-md border border-border">
          {values.map((value, i) => (
            <li key={i} className="flex gap-3 px-3 py-1.5 font-mono text-sm">
              <span className="shrink-0 select-none text-muted-foreground">
                {i + 1})
              </span>
              <span className="min-w-0 whitespace-pre-wrap break-all">
                {value}
              </span>
            </li>
          ))}
        </ol>
      )}
      <p className="text-xs text-muted-foreground">
        {result.command} · {result.rowCount}{" "}
        {result.rowCount === 1 ? "row" : "rows"}
        {result.truncated && (
          <span> · first {result.rows.length} shown (truncated)</span>
        )}{" "}
        · {result.durationMs} ms
      </p>
    </div>
  );
}

/** A single scalar reply, in a scrollable mono block (NULL muted). */
function ValueBlock({ value }: { value: string | null }) {
  if (value === null) {
    return <span className="text-sm italic text-muted-foreground">null</span>;
  }
  return (
    <pre className="max-h-72 overflow-auto whitespace-pre-wrap break-all rounded-md border border-border bg-muted/30 p-3 font-mono text-sm">
      {value}
    </pre>
  );
}

/** Stringify a reply cell: null stays null, arrays/objects become JSON. */
function formatCell(value: unknown): string | null {
  if (value === null || value === undefined) return null;
  if (typeof value === "object") return JSON.stringify(value);
  return String(value);
}

/**
 * Build the follow-up SCAN command from the one that was run, swapping in the
 * returned cursor and keeping any MATCH/COUNT/TYPE options intact.
 */
function nextScanCommand(command: string, cursor: string): string {
  const parts = command.trim().split(/\s+/);
  parts[0] = "SCAN";
  if (parts.length >= 2) {
    parts[1] = cursor;
  } else {
    parts.push(cursor);
  }
  return parts.join(" ");
}
