import { AlertCircle, Play } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useAsync } from "@/hooks/useAsync";
import { executeQuery, getSchema } from "@/services/api";
import type { QueryResult } from "@/types/console";

import { QueryResultTable } from "./QueryResultTable";
import { SqlEditor } from "./SqlEditor";

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
 * Statement failures come back inside the response envelope, so they render
 * as an expected outcome, not a transport error.
 */
export function PostgresConsole({ profile, service }: PostgresConsoleProps) {
  const [sql, setSql] = useState("");
  const [run, setRun] = useState<RunState>({ status: "idle" });

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

  // The console keeps its own controller (rather than useAsync) because runs
  // are user-triggered, not mount-triggered.
  const controllerRef = useRef<AbortController | null>(null);
  useEffect(() => () => controllerRef.current?.abort(), []);

  const runQuery = useCallback(() => {
    const statement = sql.trim();
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
          onChange={setSql}
          schema={completionSchema}
          onRun={runQuery}
        />
        <div className="flex items-center justify-between gap-3">
          <p className="text-xs text-muted-foreground">
            <kbd className="rounded border border-border px-1 font-mono">
              ⌘↵
            </kbd>{" "}
            to run
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
  );
}
