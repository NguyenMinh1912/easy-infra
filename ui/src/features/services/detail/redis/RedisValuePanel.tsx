import { AlertCircle, Inbox, Search } from "lucide-react";
import { useMemo, useState, type ReactNode } from "react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import { useAsync } from "@/hooks/useAsync";
import { getKeyValue } from "@/services/api";
import type { KeyValue } from "@/types/redis";

import { CopyButton } from "./CopyButton";
import { RedisTypeBadge } from "./RedisTypeBadge";

interface RedisValuePanelProps {
  profile: string;
  service: string;
  db: number;
  /** Key to display; null when nothing is selected yet. */
  selectedKey: string | null;
}

/** A copyable cell within a collection row (e.g. a hash field or value). */
interface RowCopy {
  label: string;
  value: string;
}

/** One rendered row: the visible cells plus what its hover actions copy. */
interface Row {
  cells: string[];
  copies: RowCopy[];
}

/** Fills the panel's height so empty/error/loading states match the data view. */
function PanelShell({ children }: { children: ReactNode }) {
  return <div className="flex h-full min-h-0 flex-col">{children}</div>;
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
      <PanelShell>
        <div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-3 rounded-md border border-dashed border-border p-10 text-center">
          <span className="flex size-10 items-center justify-center rounded-lg bg-muted">
            <Inbox className="size-5 text-muted-foreground" aria-hidden />
          </span>
          <p className="text-sm text-muted-foreground">
            Select a key to view its value.
          </p>
        </div>
      </PanelShell>
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
    return (
      <PanelShell>
        <Skeleton className="min-h-0 flex-1" />
      </PanelShell>
    );
  }
  if (state.status === "error") {
    return (
      <PanelShell>
        <Alert variant="destructive">
          <AlertCircle />
          <div>
            <AlertTitle>Could not load value</AlertTitle>
            <AlertDescription>{state.error.message}</AlertDescription>
          </div>
        </Alert>
      </PanelShell>
    );
  }
  if (state.data.error) {
    return (
      <PanelShell>
        <Alert variant="destructive">
          <AlertCircle />
          <div>
            <AlertTitle>Redis unreachable</AlertTitle>
            <AlertDescription className="font-mono text-xs">
              {state.data.error}
            </AlertDescription>
          </div>
        </Alert>
      </PanelShell>
    );
  }

  const value = state.data;
  if (value.type === "none") {
    return (
      <PanelShell>
        <div className="flex min-h-0 flex-1 items-center justify-center rounded-md border border-dashed border-border p-10 text-center">
          <p className="text-sm text-muted-foreground">
            Key <span className="font-mono">{selectedKey}</span> no longer
            exists.
          </p>
        </div>
      </PanelShell>
    );
  }

  return (
    <PanelShell>
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <span className="min-w-0 truncate font-mono text-sm font-medium">
          {value.key}
        </span>
        <CopyButton value={value.key} label="key" className="size-6" />
        <RedisTypeBadge type={value.type} />
        <Badge variant="outline">{formatTtl(value.ttl)}</Badge>
        <span className="text-xs text-muted-foreground">
          {value.length} {value.length === 1 ? "element" : "elements"}
        </span>
      </div>
      <ValueContent value={value} />
      {value.truncated && (
        <p className="mt-2 text-xs text-muted-foreground">
          Showing the first {shownCount(value)} of {value.length} — value
          truncated.
        </p>
      )}
    </PanelShell>
  );
}

/** Renders the value body according to its type. */
function ValueContent({ value }: { value: KeyValue }) {
  switch (value.type) {
    case "string":
      return <StringValue value={value.string ?? ""} />;
    case "list":
      return (
        <CollectionTable
          columns={["#", "value"]}
          rows={(value.list ?? []).map((item, i) => ({
            cells: [String(i), item],
            copies: [{ label: "value", value: item }],
          }))}
        />
      );
    case "set":
      return (
        <CollectionTable
          columns={["member"]}
          rows={(value.set ?? []).map((item) => ({
            cells: [item],
            copies: [{ label: "member", value: item }],
          }))}
        />
      );
    case "hash":
      return (
        <CollectionTable
          columns={["field", "value"]}
          rows={(value.hash ?? []).map((h) => ({
            cells: [h.field, h.value],
            copies: [
              { label: "field", value: h.field },
              { label: "value", value: h.value },
            ],
          }))}
        />
      );
    case "zset":
      return (
        <CollectionTable
          columns={["member", "score"]}
          rows={(value.zset ?? []).map((z) => ({
            cells: [z.member, String(z.score)],
            copies: [{ label: "member", value: z.member }],
          }))}
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

/** A scrollable string value with a copy action. */
function StringValue({ value }: { value: string }) {
  return (
    <div className="relative min-h-0 flex-1 rounded-md border border-border bg-muted/40">
      <CopyButton
        value={value}
        label="value"
        className="absolute top-1 right-1 z-10 size-6 bg-background/70"
      />
      <pre className="h-full overflow-auto p-3 pr-9 font-mono text-sm break-all whitespace-pre-wrap">
        {value}
      </pre>
    </div>
  );
}

/**
 * A compact table for collection values (hash, zset, list, set) with a
 * client-side filter and per-row copy actions. Columns whose every cell is
 * numeric are right-aligned with tabular figures so codes line up by digit.
 */
function CollectionTable({
  columns,
  rows,
}: {
  columns: string[];
  rows: Row[];
}) {
  const [query, setQuery] = useState("");

  const numericCol = useMemo(
    () =>
      columns.map(
        (_, col) => rows.length > 0 && rows.every((r) => isNumeric(r.cells[col])),
      ),
    [columns, rows],
  );

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return rows;
    return rows.filter((r) =>
      r.cells.some((c) => c.toLowerCase().includes(q)),
    );
  }, [rows, query]);

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-2">
      <div className="relative">
        <Search
          className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground"
          aria-hidden
        />
        <Input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder={`Filter ${columns.join(" / ")}…`}
          aria-label="Filter fields"
          className="h-8 pl-8 font-mono text-sm"
        />
      </div>

      <div className="min-h-0 flex-1 overflow-auto rounded-md border border-border">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border">
              {columns.map((c, i) => (
                <th
                  key={c}
                  className={cn(
                    "sticky top-0 z-10 bg-background px-3 py-1.5 text-left font-mono text-xs font-medium tracking-wide text-muted-foreground",
                    numericCol[i] && "text-right",
                  )}
                >
                  {c}
                </th>
              ))}
              <th className="sticky top-0 z-10 w-0 bg-background" />
            </tr>
          </thead>
          <tbody>
            {filtered.map((row, i) => (
              <tr
                key={i}
                className="group border-b border-border last:border-0 hover:bg-muted/50"
              >
                {row.cells.map((cell, j) => (
                  <td
                    key={j}
                    className={cn(
                      "px-3 py-1 align-top font-mono break-all",
                      numericCol[j] && "text-right tabular-nums",
                    )}
                  >
                    {cell}
                  </td>
                ))}
                <td className="w-0 py-0 pr-1 align-middle">
                  <div className="flex justify-end gap-0.5 opacity-0 group-hover:opacity-100 focus-within:opacity-100">
                    {row.copies.map((c) => (
                      <CopyButton
                        key={c.label}
                        value={c.value}
                        label={c.label}
                        className="size-6"
                      />
                    ))}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {filtered.length === 0 && (
          <p className="p-6 text-center text-sm text-muted-foreground">
            Nothing matches “{query}”.
          </p>
        )}
      </div>
    </div>
  );
}

/** True for a non-empty integer or decimal string (used for column alignment). */
function isNumeric(s: string): boolean {
  return s.trim() !== "" && /^-?\d+(\.\d+)?$/.test(s.trim());
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
