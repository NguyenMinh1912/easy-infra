import { useCallback, useEffect, useRef, useState } from "react";
import { AlertCircle } from "lucide-react";

import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Skeleton } from "@/components/ui/skeleton";
import { executeQuery } from "@/services/api";
import type { QueryResult } from "@/types/console";

import { QueryResultTable } from "../console/QueryResultTable";
import type { TableRef } from "./useRelationGraph";

/** Rows the data panel previews for a table. */
const PREVIEW_LIMIT = 50;

/** Quote a SQL identifier, doubling any embedded quotes. */
function quoteIdent(name: string): string {
  return `"${name.replace(/"/g, '""')}"`;
}

/** The statement the panel runs to preview a table's rows. */
function previewStatement(schema: string, table: string): string {
  return `SELECT * FROM ${quoteIdent(schema)}.${quoteIdent(table)} LIMIT ${PREVIEW_LIMIT};`;
}

interface NodeDataSheetProps {
  /** The table to preview, or null when the panel is closed. */
  table: TableRef | null;
  profile: string;
  service: string;
  onOpenChange: (open: boolean) => void;
}

type FetchState =
  | { status: "idle" }
  | { status: "running" }
  | { status: "done"; result: QueryResult }
  | { status: "failed"; error: string };

/**
 * Side panel showing a table node's data: the auto-generated relation query and
 * a preview of the rows it returns. The result is the same one the SQL console
 * renders, so its cells edit in place and its rows can be explored through their
 * own relations.
 */
export function NodeDataSheet({
  table,
  profile,
  service,
  onOpenChange,
}: NodeDataSheetProps) {
  const [state, setState] = useState<FetchState>({ status: "idle" });
  const statement = table ? previewStatement(table.schema, table.table) : "";

  const controllerRef = useRef<AbortController | null>(null);
  useEffect(() => () => controllerRef.current?.abort(), []);

  const run = useCallback(
    (sql: string) => {
      controllerRef.current?.abort();
      const controller = new AbortController();
      controllerRef.current = controller;
      setState({ status: "running" });
      executeQuery(profile, service, sql, controller.signal)
        .then((result) => {
          if (controller.signal.aborted) return;
          if (result.error) {
            setState({ status: "failed", error: result.error });
          } else {
            setState({ status: "done", result });
          }
        })
        .catch((cause: unknown) => {
          if (controller.signal.aborted) return;
          setState({
            status: "failed",
            error: cause instanceof Error ? cause.message : String(cause),
          });
        });
    },
    [profile, service],
  );

  // Re-run whenever the selected table changes (keyed by the statement, which
  // is derived from it).
  useEffect(() => {
    if (statement) run(statement);
  }, [statement, run]);

  return (
    <Sheet open={table !== null} onOpenChange={onOpenChange}>
      <SheetContent
        side="right"
        className="flex w-full flex-col gap-4 overflow-y-auto sm:max-w-2xl"
      >
        <SheetHeader>
          <SheetTitle className="font-mono">
            {table ? `${table.schema}.${table.table}` : ""}
          </SheetTitle>
          <SheetDescription>
            The relation query for this table and a preview of its rows.
          </SheetDescription>
        </SheetHeader>

        <pre className="overflow-x-auto rounded-md border border-border bg-muted/30 p-3 font-mono text-xs">
          {statement}
        </pre>

        {state.status === "running" && (
          <div className="space-y-2" aria-label="Loading rows">
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-2/3" />
          </div>
        )}
        {state.status === "failed" && (
          <div className="flex items-center gap-2 rounded-md border border-destructive/50 p-3 text-sm text-destructive">
            <AlertCircle className="size-4 shrink-0" aria-hidden />
            <span className="font-mono text-xs">{state.error}</span>
          </div>
        )}
        {state.status === "done" && (
          <QueryResultTable
            result={state.result}
            profile={profile}
            service={service}
            onChanged={() => run(statement)}
          />
        )}
      </SheetContent>
    </Sheet>
  );
}
