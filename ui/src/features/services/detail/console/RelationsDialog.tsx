import { useCallback, useEffect, useRef, useState } from "react";
import { AlertCircle, ArrowDownLeft, ArrowUpRight } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import { relatedRows, type RelationFilter } from "@/services/api";
import type { QueryResult, Relation } from "@/types/console";

import { QueryResultTable } from "./QueryResultTable";

interface RelationsDialogProps {
  /** Foreign-key paths available from the originating row's table. */
  relations: Relation[];
  /**
   * The originating row's value for one of its columns, as text (null for SQL
   * NULL), or undefined when that column isn't present in the result — in which
   * case the relation that needs it can't be followed.
   */
  localValue: (column: string) => string | null | undefined;
  /** Profile the originating result was queried from. */
  profile: string;
  /** Service the originating result was queried from. */
  service: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

/** State of the related-rows fetch for the currently selected relation. */
type FetchState =
  | { status: "idle" }
  | { status: "running" }
  | { status: "done"; result: QueryResult }
  | { status: "failed"; error: string };

/**
 * Explore the data related to one row through its table's foreign keys. The
 * user picks a relation — a parent it references or children that reference it
 * — and the rows on the far side load into a result table. That table is the
 * same one the console uses, so its rows are editable and carry their own
 * relations, letting the user keep following the graph.
 */
export function RelationsDialog({
  relations,
  localValue,
  profile,
  service,
  open,
  onOpenChange,
}: RelationsDialogProps) {
  const [selected, setSelected] = useState<Relation | null>(null);
  const [state, setState] = useState<FetchState>({ status: "idle" });

  const controllerRef = useRef<AbortController | null>(null);
  useEffect(() => () => controllerRef.current?.abort(), []);

  // The equality predicates joining a relation's table to the originating row,
  // or null when a column the relation needs isn't present in the result.
  const filtersFor = useCallback(
    (relation: Relation): RelationFilter[] | null => {
      const filters: RelationFilter[] = [];
      for (const col of relation.columns) {
        const value = localValue(col.local);
        if (value === undefined) return null;
        filters.push({ column: col.foreign, value });
      }
      return filters;
    },
    [localValue],
  );

  const load = useCallback(
    (relation: Relation) => {
      const filters = filtersFor(relation);
      if (!filters) return;
      controllerRef.current?.abort();
      const controller = new AbortController();
      controllerRef.current = controller;
      setSelected(relation);
      setState({ status: "running" });

      relatedRows(
        profile,
        service,
        { schema: relation.schema, table: relation.table, filters },
        controller.signal,
      )
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
    [filtersFor, profile, service],
  );

  // Reset the selection each time the dialog opens so a freshly clicked row
  // starts from its relation list rather than the previous row's result.
  useEffect(() => {
    if (open) {
      setSelected(null);
      setState({ status: "idle" });
    }
  }, [open]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] max-w-4xl overflow-auto">
        <DialogHeader>
          <DialogTitle>Related data</DialogTitle>
          <DialogDescription>
            Follow a foreign key to the rows this one connects to.
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-wrap gap-2">
          {relations.map((relation) => {
            const followable = filtersFor(relation) !== null;
            const isSelected =
              selected?.constraint === relation.constraint &&
              selected?.direction === relation.direction;
            return (
              <Button
                key={`${relation.constraint}:${relation.direction}`}
                type="button"
                size="sm"
                variant={isSelected ? "default" : "outline"}
                disabled={!followable}
                title={
                  followable
                    ? joinSummary(relation)
                    : "A column this relation needs isn't in the result"
                }
                className="h-auto flex-col items-start gap-0.5 py-1.5"
                onClick={() => load(relation)}
              >
                <span className="flex items-center gap-1.5 font-mono text-xs">
                  {relation.direction === "references" ? (
                    <ArrowUpRight aria-hidden className="size-3.5" />
                  ) : (
                    <ArrowDownLeft aria-hidden className="size-3.5" />
                  )}
                  {relation.schema}.{relation.table}
                </span>
                <span
                  className={cn(
                    "text-[10px] font-normal",
                    isSelected
                      ? "text-primary-foreground/80"
                      : "text-muted-foreground",
                  )}
                >
                  {relation.direction === "references"
                    ? "references"
                    : "referenced by"}
                </span>
              </Button>
            );
          })}
        </div>

        {state.status === "running" && (
          <div className="space-y-2" aria-label="Loading related rows">
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
            onChanged={() => selected && load(selected)}
          />
        )}
      </DialogContent>
    </Dialog>
  );
}

/** Human-readable join summary for a relation's tooltip, e.g. "a = b, c = d". */
function joinSummary(relation: Relation): string {
  return relation.columns
    .map((c) => `${c.local} = ${relation.table}.${c.foreign}`)
    .join(", ");
}
