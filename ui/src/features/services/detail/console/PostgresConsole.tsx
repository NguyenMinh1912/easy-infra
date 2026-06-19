import { EditorView } from "@uiw/react-codemirror";
import { AlertCircle, Play } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useResizableHeight } from "@/hooks/useResizableHeight";
import { executeQuery } from "@/services/api";
import type { QueryResult } from "@/types/console";

import { QueryResultTable } from "./QueryResultTable";
import { SqlEditor } from "./SqlEditor";
import { statementToRun } from "./statement";

interface PostgresConsoleProps {
  /** Profile whose saved connection config the statements run against. */
  profile: string;
  /** Service name within the profile (the API path segment). */
  service: string;
  /** Current editor text — owned by the parent so it persists across tabs. */
  sql: string;
  /** Notified as the user edits the statement buffer. */
  onSqlChange: (sql: string) => void;
  /**
   * Tables/columns for editor autocomplete, keyed by table name. Owned by the
   * tabs host (shared across consoles); absent while the schema loads or when
   * introspection failed.
   */
  completionSchema?: Record<string, string[]>;
  /** Whether schema introspection has finished (drives the help text). */
  schemaResolved: boolean;
  /**
   * Run the statement once as soon as the console mounts — used when a console
   * is opened pre-filled (e.g. double-clicking a table). Cleared via
   * {@link onAutoRun} so it fires only on the initial open.
   */
  autoRun?: boolean;
  /** Called after the initial auto-run is kicked off, to clear the flag. */
  onAutoRun?: () => void;
}

/** State of the current (or last) statement execution. */
type RunState =
  | { status: "idle" }
  | { status: "running" }
  // `statement` is the exact SQL that produced this result, so inline edits can
  // re-run it to refresh — independent of where the cursor sits afterwards.
  | { status: "done"; result: QueryResult; statement: string }
  | { status: "failed"; error: string };

/**
 * SQL console against one profile's postgres: an editor with keyword and
 * schema-aware completion, a run action, and the last execution's result.
 * The editor may hold several `;`-separated statements; a run executes just
 * one — the selection if there is one, otherwise the statement under the
 * cursor. Statement failures come back inside the response envelope, so they
 * render as an expected outcome, not a transport error.
 */
export function PostgresConsole({
  profile,
  service,
  sql,
  onSqlChange,
  completionSchema,
  schemaResolved,
  autoRun,
  onAutoRun,
}: PostgresConsoleProps) {
  const [run, setRun] = useState<RunState>({ status: "idle" });

  // The editor's height is user-draggable (handle on its bottom edge) and
  // remembered across reloads. Shrinking it grows the result table below, which
  // re-caps itself to the remaining viewport — so one handle resizes both.
  const { height: editorHeight, onResizeStart } = useResizableHeight({
    key: "console-editor-height",
    initial: 224,
    min: 120,
    max: 640,
  });

  // The live editor view, used to read the cursor/selection when choosing
  // which statement to run.
  const viewRef = useRef<EditorView | null>(null);

  // The console keeps its own controller (rather than useAsync) because runs
  // are user-triggered, not mount-triggered.
  const controllerRef = useRef<AbortController | null>(null);
  useEffect(() => () => controllerRef.current?.abort(), []);

  // Execute one specific statement, replacing any in-flight run. Used both for
  // user-triggered runs and to refresh after an inline row edit.
  const execute = useCallback(
    (statement: string) => {
      if (!statement) return;
      controllerRef.current?.abort();
      const controller = new AbortController();
      controllerRef.current = controller;
      setRun({ status: "running" });

      executeQuery(profile, service, statement, controller.signal)
        .then((result) => {
          if (controller.signal.aborted) return;
          if (result.error) {
            setRun({ status: "failed", error: result.error });
          } else {
            setRun({ status: "done", result, statement });
          }
        })
        .catch((cause: unknown) => {
          if (controller.signal.aborted) return;
          setRun({
            status: "failed",
            error: cause instanceof Error ? cause.message : String(cause),
          });
        });
    },
    [profile, service],
  );

  const runQuery = useCallback(() => {
    // Run the selection if there is one, otherwise the statement under the
    // cursor; fall back to the whole buffer before the editor mounts.
    const view = viewRef.current;
    const statement = view
      ? statementToRun(view.state.doc.toString(), view.state.selection.main)
      : sql.trim();
    execute(statement);
  }, [execute, sql]);

  // Fire the initial run for a pre-filled console. The buffer holds a single
  // statement, so runQuery falls back to it even before the editor view (and
  // viewRef) is ready. onAutoRun then flips `autoRun` off so this only fires on
  // the initial open, not on later re-renders (e.g. when editing the buffer
  // changes runQuery's identity). Crucially we do NOT guard with a "ran once"
  // ref: under React StrictMode the mount runs setup→cleanup→setup, and the
  // cleanup aborts the in-flight request — a persistent ref would block the
  // second setup from re-running it, leaving the result stuck on "running".
  useEffect(() => {
    if (autoRun) {
      runQuery();
      onAutoRun?.();
    }
  }, [autoRun, runQuery, onAutoRun]);

  const running = run.status === "running";

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <div className="relative">
          <SqlEditor
            value={sql}
            onChange={onSqlChange}
            schema={completionSchema}
            onRun={runQuery}
            viewRef={viewRef}
            height={editorHeight}
          />
          <div
            role="separator"
            aria-orientation="horizontal"
            aria-label="Resize SQL editor"
            onPointerDown={onResizeStart}
            className="absolute inset-x-0 -bottom-2 h-4 cursor-row-resize touch-none after:absolute after:inset-x-0 after:top-1/2 after:h-px after:-translate-y-1/2 after:bg-transparent hover:after:bg-border"
          />
        </div>
        <div className="flex items-center justify-between gap-3">
          <p className="text-xs text-muted-foreground">
            <kbd className="rounded border border-border px-1 font-mono">
              ⌘↵
            </kbd>{" "}
            runs the statement at the cursor, or the selection
            {schemaResolved &&
              (completionSchema
                ? " · table & column suggestions on"
                : " · schema unavailable, keyword suggestions only")}
          </p>
          <Button
            size="sm"
            onClick={runQuery}
            disabled={running || sql.trim() === ""}
          >
            <Play aria-hidden /> Run query
          </Button>
        </div>
      </div>

      {run.status === "running" && (
        <div className="space-y-2" aria-label="Running query">
          <Skeleton className="h-8 w-full" />
          <Skeleton className="h-8 w-full" />
          <Skeleton className="h-8 w-2/3" />
        </div>
      )}
      {run.status === "failed" && (
        <Alert variant="destructive">
          <AlertCircle />
          <div>
            <AlertTitle>Query failed</AlertTitle>
            <AlertDescription className="font-mono text-xs">
              {run.error}
            </AlertDescription>
          </div>
        </Alert>
      )}
      {run.status === "done" && (
        <QueryResultTable
          result={run.result}
          profile={profile}
          service={service}
          onChanged={() => execute(run.statement)}
        />
      )}
    </div>
  );
}
