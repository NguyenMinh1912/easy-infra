import { useState } from "react";
import {
  Check,
  CheckCircle2,
  Expand,
  Trash2,
  Waypoints,
  X,
} from "lucide-react";
import { toast } from "sonner";

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
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useRemainingHeight } from "@/hooks/useRemainingHeight";
import { cn } from "@/lib/utils";
import { ApiError, deleteRow, updateRow, type RowKey } from "@/services/api";
import type { EditableInfo, QueryResult } from "@/types/console";

import { RelationsDialog } from "./RelationsDialog";

/** Above this many characters a cell is collapsed behind a preview dialog. */
const MAX_CELL_LENGTH = 120;

interface QueryResultTableProps {
  result: QueryResult;
  /** Profile the result was queried from — used to apply inline edits. */
  profile: string;
  /** Service the result was queried from — used to apply inline edits. */
  service: string;
  /** Re-run the statement that produced this result, e.g. after an edit. */
  onChanged: () => void;
}

/**
 * A staged inline mutation awaiting confirmation: setting one cell (update) or
 * removing a row (delete). `preview` is the human-readable statement shown in
 * the confirm dialog — illustrative, since the server runs a parameterized
 * equivalent.
 */
type Pending =
  | {
      kind: "update";
      column: string;
      key: RowKey;
      value: string | null;
      preview: string;
    }
  | { kind: "delete"; key: RowKey; preview: string };

/**
 * Renders a successful console execution: a column/row table for statements
 * that produced rows, a command-tag confirmation otherwise, and a status line
 * with the command tag, row count, duration, and a truncation notice.
 *
 * When the result maps back to a single table with a primary key (see
 * {@link QueryResult.editable}), cells become click-to-edit and each row gains a
 * delete action; both confirm before running and refresh the result on success.
 */
export function QueryResultTable({
  result,
  profile,
  service,
  onChanged,
}: QueryResultTableProps) {
  // Cap the table to the height still visible below it so a large result set
  // scrolls in place instead of pushing the page past the viewport.
  const { ref, maxHeight } = useRemainingHeight<HTMLDivElement>();

  const editable = result.editable;
  // The cell currently in edit mode, or null. Only set for editable results.
  const [editing, setEditing] = useState<{ row: number; col: number } | null>(
    null,
  );
  // A staged mutation awaiting confirmation in the dialog, or null.
  const [pending, setPending] = useState<Pending | null>(null);
  const [busy, setBusy] = useState(false);
  // The row whose relations are being explored in the dialog, or null.
  const [relationRow, setRelationRow] = useState<number | null>(null);

  const relations = editable?.relations ?? [];

  // The relations dialog resolves a source-table column to its value in the
  // row being explored, or undefined when that column isn't in the result (so
  // a relation needing it can't be followed).
  const localValue = (column: string): string | null | undefined => {
    if (!editable || relationRow === null) return undefined;
    const j = editable.columns.indexOf(column);
    if (j < 0) return undefined;
    const value = result.rows[relationRow][j];
    return isNull(value) ? null : formatValue(value);
  };

  // The primary-key values identifying row `i`, or null if any key column is
  // missing (then the row can't be edited and its affordances are hidden).
  const rowKey = (i: number): RowKey | null => {
    if (!editable) return null;
    const key: RowKey = {};
    for (const pk of editable.primaryKey) {
      const j = editable.columns.indexOf(pk);
      if (j < 0) return null;
      key[pk] = keyText(result.rows[i][j]);
    }
    return key;
  };

  // Stage a cell update for confirmation, skipping no-op edits.
  const stageUpdate = (i: number, j: number, value: string | null) => {
    setEditing(null);
    if (!editable) return;
    const column = editable.columns[j];
    const key = rowKey(i);
    if (!column || !key) return;
    const current = result.rows[i][j];
    const unchanged =
      value === null ? current === null : !isNull(current) && cellText(current) === value;
    if (unchanged) return;
    setPending({
      kind: "update",
      column,
      key,
      value,
      preview: updatePreview(editable, column, value, key),
    });
  };

  // Stage a row delete for confirmation.
  const stageDelete = (i: number) => {
    if (!editable) return;
    const key = rowKey(i);
    if (!key) return;
    setPending({ kind: "delete", key, preview: deletePreview(editable, key) });
  };

  // Run the staged mutation, then refresh the result so the table reflects it.
  const apply = async () => {
    if (!pending || !editable) return;
    const target = { schema: editable.schema, table: editable.table };
    setBusy(true);
    try {
      const res =
        pending.kind === "update"
          ? await updateRow(profile, service, {
              ...target,
              key: pending.key,
              column: pending.column,
              value: pending.value,
            })
          : await deleteRow(profile, service, { ...target, key: pending.key });
      toast.success(res.command);
      setPending(null);
      onChanged();
    } catch (cause) {
      toast.error(pending.kind === "update" ? "Update failed" : "Delete failed", {
        description: cause instanceof ApiError ? cause.message : String(cause),
      });
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="space-y-2">
      {result.columns.length > 0 ? (
        // The outer container owns both scroll axes so the horizontal
        // scrollbar stays pinned to its (height-capped) bottom edge. Neutralize
        // the inner overflow-x the shadcn Table adds, which would otherwise park
        // the x-scrollbar below the last row — only reachable after scrolling y.
        <div
          ref={ref}
          style={{ maxHeight: maxHeight ?? undefined }}
          className="overflow-auto rounded-md border border-border [&>div]:overflow-x-visible"
        >
          <Table>
            <TableHeader>
              <TableRow>
                {result.columns.map((column, i) => (
                  <TableHead
                    key={i}
                    className="sticky top-0 z-10 bg-background font-mono shadow-[inset_0_-1px_0_var(--border)]"
                  >
                    {column}
                  </TableHead>
                ))}
                {editable && (
                  <TableHead
                    className="sticky top-0 z-10 w-0 bg-background shadow-[inset_0_-1px_0_var(--border)]"
                    aria-label="Row actions"
                  />
                )}
              </TableRow>
            </TableHeader>
            <TableBody>
              {result.rows.map((row, i) => (
                <TableRow key={i}>
                  {row.map((value, j) => {
                    const column = editable?.columns[j];
                    const isEditing = editing?.row === i && editing.col === j;
                    return (
                      <TableCell key={j} className="font-mono">
                        {isEditing ? (
                          <CellEditor
                            initial={cellEditText(value)}
                            onCommit={(text) => stageUpdate(i, j, text)}
                            onNull={() => stageUpdate(i, j, null)}
                            onCancel={() => setEditing(null)}
                          />
                        ) : column ? (
                          <button
                            type="button"
                            title="Click to edit"
                            className="-mx-1 block w-full rounded px-1 text-left hover:bg-muted/60"
                            onClick={() => setEditing({ row: i, col: j })}
                          >
                            <CellValue value={value} column={result.columns[j]} />
                          </button>
                        ) : (
                          <CellValue value={value} column={result.columns[j]} />
                        )}
                      </TableCell>
                    );
                  })}
                  {editable && (
                    <TableCell className="w-0 pr-2">
                      <div className="flex items-center justify-end gap-1">
                        {relations.length > 0 && (
                          <Button
                            type="button"
                            variant="ghost"
                            size="icon"
                            className="size-7 text-muted-foreground hover:text-primary"
                            aria-label="Show related data"
                            title="Show related data"
                            onClick={() => setRelationRow(i)}
                          >
                            <Waypoints aria-hidden />
                          </Button>
                        )}
                        {rowKey(i) && (
                          <Button
                            type="button"
                            variant="ghost"
                            size="icon"
                            className="size-7 text-muted-foreground hover:text-destructive"
                            aria-label="Delete row"
                            onClick={() => stageDelete(i)}
                          >
                            <Trash2 aria-hidden />
                          </Button>
                        )}
                      </div>
                    </TableCell>
                  )}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      ) : (
        <div className="flex items-center gap-2 rounded-md border border-border p-4 text-sm">
          <CheckCircle2 className="size-4 text-success" aria-hidden />
          <span className="font-mono">{result.command}</span>
        </div>
      )}
      <p className="text-xs text-muted-foreground">
        {result.command} · {result.rowCount}{" "}
        {result.rowCount === 1 ? "row" : "rows"}
        {result.truncated && (
          <span> · showing first {result.rows.length} rows (truncated)</span>
        )}{" "}
        · {result.durationMs} ms
        {editable && <span> · click a cell to edit · rows deletable</span>}
        {relations.length > 0 && <span> · explore related rows</span>}
      </p>

      <AlertDialog
        open={pending !== null}
        onOpenChange={(open) => {
          if (!open && !busy) setPending(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {pending?.kind === "delete" ? "Delete this row?" : "Apply this update?"}
            </AlertDialogTitle>
            <AlertDialogDescription>
              This runs the following statement against{" "}
              <span className="font-mono">
                {editable?.schema}.{editable?.table}
              </span>
              :
            </AlertDialogDescription>
          </AlertDialogHeader>
          <pre className="max-h-[40vh] overflow-auto whitespace-pre-wrap break-words rounded-md border border-border bg-muted/30 p-3 font-mono text-xs">
            {pending?.preview}
          </pre>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={busy}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              disabled={busy}
              className={cn(
                pending?.kind === "delete" &&
                  buttonVariants({ variant: "destructive" }),
              )}
              onClick={(e) => {
                // Keep the dialog open while the request is in flight; apply()
                // closes it on success.
                e.preventDefault();
                void apply();
              }}
            >
              {pending?.kind === "delete" ? "Delete" : "Update"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {editable && relations.length > 0 && (
        <RelationsDialog
          relations={relations}
          localValue={localValue}
          profile={profile}
          service={service}
          open={relationRow !== null}
          onOpenChange={(open) => {
            if (!open) setRelationRow(null);
          }}
        />
      )}
    </div>
  );
}

/** Inline cell editor: a text field with save, set-null, and cancel actions. */
function CellEditor({
  initial,
  onCommit,
  onNull,
  onCancel,
}: {
  initial: string;
  onCommit: (text: string) => void;
  onNull: () => void;
  onCancel: () => void;
}) {
  const [text, setText] = useState(initial);
  return (
    <div className="flex items-center gap-1">
      <Input
        autoFocus
        value={text}
        onChange={(e) => setText(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Enter") {
            e.preventDefault();
            onCommit(text);
          } else if (e.key === "Escape") {
            e.preventDefault();
            onCancel();
          }
        }}
        className="h-7 min-w-40 flex-1 font-mono text-xs"
      />
      <Button
        type="button"
        variant="ghost"
        size="icon"
        className="size-7 shrink-0"
        aria-label="Save"
        onClick={() => onCommit(text)}
      >
        <Check aria-hidden />
      </Button>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        className="h-7 shrink-0 px-2 text-xs"
        title="Set to NULL"
        onClick={onNull}
      >
        NULL
      </Button>
      <Button
        type="button"
        variant="ghost"
        size="icon"
        className="size-7 shrink-0"
        aria-label="Cancel"
        onClick={onCancel}
      >
        <X aria-hidden />
      </Button>
    </div>
  );
}

/** True when a cell value is SQL NULL (JSON null/undefined). */
function isNull(value: unknown): boolean {
  return value === null || value === undefined;
}

/** Render a cell value as text: objects (json, arrays) serialized, rest stringified. */
function formatValue(value: unknown): string {
  if (typeof value === "object") {
    return JSON.stringify(value);
  }
  return String(value);
}

/** Stringify a primary-key value for the row identity sent to the server. */
function keyText(value: unknown): string {
  return isNull(value) ? "" : formatValue(value);
}

/** Initial text for the inline editor; NULL starts as an empty field. */
function cellEditText(value: unknown): string {
  return isNull(value) ? "" : formatValue(value);
}

/** Stringify a (non-null) cell value for comparing against an edited value. */
function cellText(value: unknown): string {
  return formatValue(value);
}

/** Quote a SQL identifier for the confirmation preview, doubling quotes. */
function quoteIdent(name: string): string {
  return `"${name.replace(/"/g, '""')}"`;
}

/** Render a value as a SQL literal for the confirmation preview. */
function sqlLiteral(value: string | null): string {
  if (value === null) return "NULL";
  return `'${value.replace(/'/g, "''")}'`;
}

/** The primary-key predicate shared by the update and delete previews. */
function whereText(editable: EditableInfo, key: RowKey): string {
  return editable.primaryKey
    .map((pk) => `${quoteIdent(pk)} = ${sqlLiteral(key[pk])}`)
    .join(" AND ");
}

function updatePreview(
  editable: EditableInfo,
  column: string,
  value: string | null,
  key: RowKey,
): string {
  return `UPDATE ${quoteIdent(editable.schema)}.${quoteIdent(editable.table)}\nSET ${quoteIdent(column)} = ${sqlLiteral(value)}\nWHERE ${whereText(editable, key)};`;
}

function deletePreview(editable: EditableInfo, key: RowKey): string {
  return `DELETE FROM ${quoteIdent(editable.schema)}.${quoteIdent(editable.table)}\nWHERE ${whereText(editable, key)};`;
}

/**
 * Pretty-print text as JSON when it parses as an object or array, otherwise
 * return it unchanged so plain text previews stay as-is.
 */
function prettyFormat(text: string): string {
  const trimmed = text.trim();
  if (!trimmed.startsWith("{") && !trimmed.startsWith("[")) {
    return text;
  }
  try {
    return JSON.stringify(JSON.parse(trimmed), null, 2);
  } catch {
    return text;
  }
}

/**
 * One cell: NULL muted, long values collapsed behind a preview dialog, the rest
 * rendered inline as text.
 */
function CellValue({ value, column }: { value: unknown; column: string }) {
  if (value === null || value === undefined) {
    return <span className="italic text-muted-foreground">null</span>;
  }
  const text = formatValue(value);
  if (text.length > MAX_CELL_LENGTH) {
    return <LongCellValue text={text} column={column} />;
  }
  return <>{text}</>;
}

/** A long cell value: truncated to one line with a button to preview it in full. */
function LongCellValue({ text, column }: { text: string; column: string }) {
  const [open, setOpen] = useState(false);
  return (
    <div className="flex items-center gap-2">
      <span className="block max-w-md truncate">{text}</span>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        className="h-6 shrink-0 gap-1 px-2 text-xs"
        onClick={(e) => {
          // Don't let the click also start an inline edit on the cell.
          e.stopPropagation();
          setOpen(true);
        }}
      >
        <Expand aria-hidden />
        View
      </Button>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle className="font-mono">{column}</DialogTitle>
          </DialogHeader>
          <pre className="max-h-[60vh] overflow-auto whitespace-pre-wrap break-words rounded-md border border-border bg-muted/30 p-4 font-mono text-sm">
            {prettyFormat(text)}
          </pre>
        </DialogContent>
      </Dialog>
    </div>
  );
}
