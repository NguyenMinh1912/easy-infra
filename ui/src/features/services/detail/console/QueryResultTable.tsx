import { CheckCircle2 } from "lucide-react";

import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { QueryResult } from "@/types/console";

interface QueryResultTableProps {
  result: QueryResult;
}

/**
 * Renders a successful console execution: a column/row table for statements
 * that produced rows, a command-tag confirmation otherwise, and a status line
 * with the command tag, row count, duration, and a truncation notice.
 */
export function QueryResultTable({ result }: QueryResultTableProps) {
  return (
    <div className="space-y-2">
      {result.columns.length > 0 ? (
        <div className="overflow-x-auto rounded-md border border-border">
          <Table>
            <TableHeader>
              <TableRow>
                {result.columns.map((column, i) => (
                  <TableHead key={i} className="font-mono">
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
                      <CellValue value={value} />
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

/** One cell: NULL muted, objects (json, arrays) serialized, the rest as text. */
function CellValue({ value }: { value: unknown }) {
  if (value === null || value === undefined) {
    return <span className="italic text-muted-foreground">null</span>;
  }
  if (typeof value === "object") {
    return <>{JSON.stringify(value)}</>;
  }
  return <>{String(value)}</>;
}
