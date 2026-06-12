import { EditorView } from "@uiw/react-codemirror";
import { AlertCircle, Play } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useAsync } from "@/hooks/useAsync";
import { executeQuery, getSchema } from "@/services/api";
import type { QueryResult } from "@/types/console";

import { QueryResultTable } from "./QueryResultTable";
import { SchemaSidebar } from "./SchemaSidebar";
import { SqlEditor } from "./SqlEditor";
import { statementToRun } from "./statement";

interface PostgresConsoleProps {
  /** Profile whose saved connection config the statements run against. */
  profile: string;
  /** Service name within the profile (the API path segment). */
  service: string;
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
export function PostgresConsole({ profile, service }: PostgresConsoleProps) {
  const [sql, setSql] = useState("");
  const [run, setRun] = useState<RunState>({ status: "idle" });
  // The schema browsed in the sidebar. Defaults to the connection's configured
  // schema once introspection lands (see the effect below).
  const [selectedSchema, setSelectedSchema] = useState<string | null>(null);

  // The live editor view, used to read the cursor/selection when choosing
  // which statement to run.
  const viewRef = useRef<EditorView | null>(null);

  // Fetched once per mount; completion degrades to keywords-only while it
  // loads or when introspection fails.
  const schemaState = useAsync(
    (signal) => getSchema(profile, service, signal),
    [profile, service],
  );
  const completionSchema = useMemo(() => {
    if (schemaState.status !== "success" || schemaState.data.error) {
      return undefined;
    }
    // Tables in the connection's current schema (its search_path) complete
    // unqualified, matching how unqualified names resolve when the statement
    // runs; tables in other schemas keep their schema prefix.
    const current = schemaState.data.currentSchema || "public";
    const schema: Record<string, string[]> = {};
    for (const table of schemaState.data.tables) {
      const key =
        table.schema === current
          ? table.name
          : `${table.schema}.${table.name}`;
      schema[key] = table.columns;
    }
    return schema;
  }, [schemaState]);

  // Schema introspection lands either with usable data or an `error` envelope
  // (database unreachable); the sidebar mirrors the latter as "unavailable".
  const schemaInfo =
    schemaState.status === "success" && !schemaState.data.error
      ? schemaState.data
      : null;

  // Distinct schemas the connection can see, plus the configured one even if it
  // holds no tables, so the default selection always appears in the dropdown.
  const schemas = useMemo(() => {
    if (!schemaInfo) return [];
    const names = new Set<string>();
    for (const table of schemaInfo.tables) names.add(table.schema);
    if (schemaInfo.currentSchema) names.add(schemaInfo.currentSchema);
    return Array.from(names).sort();
  }, [schemaInfo]);

  // Default the sidebar to the connection's configured schema; reselect when it
  // changes (a different profile/service was navigated to).
  const currentSchema = schemaInfo?.currentSchema || null;
  useEffect(() => {
    if (currentSchema) setSelectedSchema(currentSchema);
  }, [currentSchema]);

  const tablesInSchema = useMemo(() => {
    if (!schemaInfo || !selectedSchema) return [];
    return schemaInfo.tables.filter((t) => t.schema === selectedSchema);
  }, [schemaInfo, selectedSchema]);

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
    <div className="flex gap-4">
      <SchemaSidebar
        loading={schemaState.status === "loading"}
        unavailable={schemaState.status === "success" && !schemaInfo}
        schemas={schemas}
        selected={selectedSchema}
        onSelect={setSelectedSchema}
        tables={tablesInSchema}
      />
      <div className="min-w-0 flex-1 space-y-4">
      <div className="space-y-2">
        <SqlEditor
          value={sql}
          onChange={setSql}
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
            {schemaState.status === "success" &&
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
    </div>
  );
}
