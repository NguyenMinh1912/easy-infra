import { useState } from "react";
import { CheckCircle2, Expand } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useRemainingHeight } from "@/hooks/useRemainingHeight";
import type { QueryResult } from "@/types/console";

/** Above this many characters a cell is collapsed behind a preview dialog. */
const MAX_CELL_LENGTH = 120;

interface QueryResultTableProps {
  result: QueryResult;
}

/**
 * Renders a successful console execution: a column/row table for statements
 * that produced rows, a command-tag confirmation otherwise, and a status line
 * with the command tag, row count, duration, and a truncation notice.
 */
export function QueryResultTable({ result }: QueryResultTableProps) {
  // Cap the table to the height still visible below it so a large result set
  // scrolls in place instead of pushing the page past the viewport.
  const { ref, maxHeight } = useRemainingHeight<HTMLDivElement>();
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
              </TableRow>
            </TableHeader>
            <TableBody>
              {result.rows.map((row, i) => (
                <TableRow key={i}>
                  {row.map((value, j) => (
                    <TableCell key={j} className="font-mono">
                      <CellValue value={value} column={result.columns[j]} />
                    </TableCell>
                  ))}
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
      </p>
    </div>
  );
}

/** Render a cell value as text: objects (json, arrays) serialized, rest stringified. */
function formatValue(value: unknown): string {
  if (typeof value === "object") {
    return JSON.stringify(value);
  }
  return String(value);
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
        onClick={() => setOpen(true)}
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
            {text}
          </pre>
        </DialogContent>
      </Dialog>
    </div>
  );
}
