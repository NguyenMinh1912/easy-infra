import { EditorView } from "@uiw/react-codemirror";
import { AlertCircle, Play } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
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
}

/** State of the current (or last) statement execution. */
type RunState =
  | { status: "idle" }
  | { status: "running" }
  | { status: "done"; result: QueryResult }
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
}: PostgresConsoleProps) {
  const [run, setRun] = useState<RunState>({ status: "idle" });

  // The live editor view, used to read the cursor/selection when choosing
  // which statement to run.
  const viewRef = useRef<EditorView | null>(null);

  // The console keeps its own controller (rather than useAsync) because runs
  // are user-triggered, not mount-triggered.
  const controllerRef = useRef<AbortController | null>(null);
  useEffect(() => () => controllerRef.current?.abort(), []);

  const runQuery = useCallback(() => {
    // Run the selection if there is one, otherwise the statement under the
    // cursor; fall back to the whole buffer before the editor mounts.
    const view = viewRef.current;
    const statement = view
      ? statementToRun(view.state.doc.toString(), view.state.selection.main)
      : sql.trim();
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
          setRun({ status: "done", result });
        }
      })
      .catch((cause: unknown) => {
        if (controller.signal.aborted) return;
        setRun({
          status: "failed",
          error: cause instanceof Error ? cause.message : String(cause),
        });
      });
  }, [profile, service, sql]);

  const running = run.status === "running";

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <SqlEditor
          value={sql}
          onChange={onSqlChange}
          schema={completionSchema}
          onRun={runQuery}
          viewRef={viewRef}
        />
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
      {run.status === "done" && <QueryResultTable result={run.result} />}
    </div>
  );
}
