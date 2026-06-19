import { AlertCircle } from "lucide-react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useAsync } from "@/hooks/useAsync";
import { getKeyValue } from "@/services/api";
import type { KeyValue } from "@/types/redis";

interface RedisValuePanelProps {
  profile: string;
  service: string;
  db: number;
  /** Key to display; null when nothing is selected yet. */
  selectedKey: string | null;
}

/**
 * Reads and renders one key's value, shaped by its Redis type. A read failure
 * (server unreachable) comes back inside the response envelope, so it renders
 * as an expected outcome rather than a transport error.
 */
export function RedisValuePanel({
  profile,
  service,
  db,
  selectedKey,
}: RedisValuePanelProps) {
  if (selectedKey === null) {
    return (
      <div className="flex h-full min-h-48 items-center justify-center rounded-md border border-dashed border-border p-10 text-center">
        <p className="text-sm text-muted-foreground">
          Select a key to view its value.
        </p>
      </div>
    );
  }
  return (
    <ValueBody
      key={`${db}:${selectedKey}`}
      profile={profile}
      service={service}
      db={db}
      selectedKey={selectedKey}
    />
  );
}

function ValueBody({
  profile,
  service,
  db,
  selectedKey,
}: RedisValuePanelProps & { selectedKey: string }) {
  const state = useAsync(
    (signal) => getKeyValue(profile, service, db, selectedKey, signal),
    [profile, service, db, selectedKey],
  );

  if (state.status === "loading") {
    return <Skeleton className="h-40 w-full" />;
  }
  if (state.status === "error") {
    return (
      <Alert variant="destructive">
        <AlertCircle />
        <div>
          <AlertTitle>Could not load value</AlertTitle>
          <AlertDescription>{state.error.message}</AlertDescription>
        </div>
      </Alert>
    );
  }
  if (state.data.error) {
    return (
      <Alert variant="destructive">
        <AlertCircle />
        <div>
          <AlertTitle>Redis unreachable</AlertTitle>
          <AlertDescription className="font-mono text-xs">
            {state.data.error}
          </AlertDescription>
        </div>
      </Alert>
    );
  }

  const value = state.data;
  if (value.type === "none") {
    return (
      <div className="rounded-md border border-dashed border-border p-10 text-center">
        <p className="text-sm text-muted-foreground">
          Key <span className="font-mono">{selectedKey}</span> no longer exists.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-2">
        <span className="min-w-0 truncate font-mono text-sm font-medium">
          {value.key}
        </span>
        <Badge variant="secondary">{value.type}</Badge>
        <Badge variant="outline">{formatTtl(value.ttl)}</Badge>
        <span className="text-xs text-muted-foreground">
          {value.length} {value.length === 1 ? "element" : "elements"}
        </span>
      </div>
      <ValueContent value={value} />
      {value.truncated && (
        <p className="text-xs text-muted-foreground">
          Showing the first {shownCount(value)} of {value.length} — value
          truncated.
        </p>
      )}
    </div>
  );
}

/** Renders the value body according to its type. */
function ValueContent({ value }: { value: KeyValue }) {
  switch (value.type) {
    case "string":
      return (
        <pre className="max-h-96 overflow-auto rounded-md border border-border bg-muted/40 p-3 font-mono text-sm whitespace-pre-wrap break-all">
          {value.string ?? ""}
        </pre>
      );
    case "list":
      return <IndexedList items={value.list ?? []} />;
    case "set":
      return <PlainList items={value.set ?? []} />;
    case "hash":
      return (
        <PairTable
          columns={["field", "value"]}
          rows={(value.hash ?? []).map((h) => [h.field, h.value])}
        />
      );
    case "zset":
      return (
        <PairTable
          columns={["member", "score"]}
          rows={(value.zset ?? []).map((z) => [z.member, String(z.score)])}
        />
      );
    default:
      return (
        <p className="rounded-md border border-border p-4 text-sm text-muted-foreground">
          Previewing <span className="font-mono">{value.type}</span> values is
          not supported yet.
        </p>
      );
  }
}

/** A list rendered with positional indices (preserves list order). */
function IndexedList({ items }: { items: string[] }) {
  return (
    <PairTable
      columns={["#", "value"]}
      rows={items.map((item, i) => [String(i), item])}
    />
  );
}

/** A single-column list (set members). */
function PlainList({ items }: { items: string[] }) {
  return (
    <div className="max-h-96 overflow-auto rounded-md border border-border">
      <Table>
        <TableBody>
          {items.map((item, i) => (
            <TableRow key={i}>
              <TableCell className="font-mono break-all">{item}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

/** A two-column table for hash fields, zset members, or indexed lists. */
function PairTable({
  columns,
  rows,
}: {
  columns: [string, string];
  rows: string[][];
}) {
  return (
    <div className="max-h-96 overflow-auto rounded-md border border-border">
      <Table>
        <TableHeader>
          <TableRow>
            {columns.map((c) => (
              <TableHead
                key={c}
                className="sticky top-0 z-10 bg-background font-mono"
              >
                {c}
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((row, i) => (
            <TableRow key={i}>
              {row.map((cell, j) => (
                <TableCell key={j} className="font-mono break-all">
                  {cell}
                </TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

/** Human-readable TTL: -1 no expiry, -2 missing, otherwise seconds. */
function formatTtl(ttl: number): string {
  if (ttl === -1) return "no expiry";
  if (ttl === -2) return "missing";
  return `TTL ${ttl}s`;
}

/** How many elements the panel actually rendered, for the truncation note. */
function shownCount(value: KeyValue): number {
  switch (value.type) {
    case "list":
      return value.list?.length ?? 0;
    case "set":
      return value.set?.length ?? 0;
    case "hash":
      return value.hash?.length ?? 0;
    case "zset":
      return value.zset?.length ?? 0;
    default:
      return 0;
  }
}
