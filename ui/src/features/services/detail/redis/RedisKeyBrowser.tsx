import {
  AlertCircle,
  ArrowDownAZ,
  ArrowUpAZ,
  ChevronsUpDown,
  Database,
  ListOrdered,
  Search,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import { useAsync } from "@/hooks/useAsync";
import { useRemainingHeight } from "@/hooks/useRemainingHeight";
import { getDatabases, listKeys } from "@/services/api";
import type { KeyEntry } from "@/types/redis";

import { CopyButton } from "./CopyButton";
import { RedisTypeBadge } from "./RedisTypeBadge";
import { RedisValuePanel } from "./RedisValuePanel";

interface RedisKeyBrowserProps {
  /** Profile whose saved connection config the scan runs against. */
  profile: string;
  /** Service name within the profile (the API path segment). */
  service: string;
}

/** Accumulated key-list state across SCAN pages. */
interface ListState {
  keys: KeyEntry[];
  cursor: number;
  status: "loading" | "more" | "done" | "error";
  error?: string;
}

/** How the loaded keys are ordered for display (scan order by default). */
type SortMode = "scan" | "asc" | "desc";

/**
 * Key browser for one profile's Redis: a logical-database selector and pattern
 * filter drive a cursor-paged SCAN, and selecting a key shows its value. A scan
 * failure (server unreachable) comes back inside the response envelope, so it
 * renders as an expected outcome rather than a transport error.
 */
export function RedisKeyBrowser({ profile, service }: RedisKeyBrowserProps) {
  const dbState = useAsync(
    (signal) => getDatabases(profile, service, signal),
    [profile, service],
  );

  const [db, setDb] = useState(0);
  // The pattern in the input vs. the one actually applied to the scan: typing
  // alone does not re-scan; submitting (Enter or the search button) does.
  const [draftPattern, setDraftPattern] = useState("*");
  const [pattern, setPattern] = useState("*");
  const [selectedKey, setSelectedKey] = useState<string | null>(null);
  const [sort, setSort] = useState<SortMode>("scan");

  const [list, setList] = useState<ListState>({
    keys: [],
    cursor: 0,
    status: "loading",
  });

  // A single in-flight scan controller, aborted when inputs change or on
  // unmount so a stale page never lands after a newer one.
  const controllerRef = useRef<AbortController | null>(null);
  useEffect(() => () => controllerRef.current?.abort(), []);

  const loadPage = useCallback(
    (cursor: number, append: boolean) => {
      controllerRef.current?.abort();
      const controller = new AbortController();
      controllerRef.current = controller;
      setList((prev) => ({
        keys: append ? prev.keys : [],
        cursor,
        status: append ? "more" : "loading",
      }));

      listKeys(profile, service, db, pattern, cursor, controller.signal)
        .then((res) => {
          if (controller.signal.aborted) return;
          if (res.error) {
            setList((prev) => ({ ...prev, status: "error", error: res.error }));
            return;
          }
          setList((prev) => ({
            keys: append ? [...prev.keys, ...res.keys] : res.keys,
            cursor: res.cursor,
            status: res.cursor === 0 ? "done" : "more",
          }));
        })
        .catch((cause: unknown) => {
          if (controller.signal.aborted) return;
          setList((prev) => ({
            ...prev,
            status: "error",
            error: cause instanceof Error ? cause.message : String(cause),
          }));
        });
    },
    [profile, service, db, pattern],
  );

  // Re-scan from the start whenever the database or applied pattern changes.
  useEffect(() => {
    loadPage(0, false);
  }, [loadPage]);

  const applyPattern = () => setPattern(draftPattern.trim() === "" ? "*" : draftPattern.trim());

  // Sort only the loaded page client-side; "scan" preserves the natural order
  // the server walked the keyspace in.
  const sortedKeys = useMemo(() => {
    if (sort === "scan") return list.keys;
    const copy = [...list.keys].sort((a, b) => a.name.localeCompare(b.name));
    return sort === "desc" ? copy.reverse() : copy;
  }, [list.keys, sort]);

  const cycleSort = () =>
    setSort((s) => (s === "scan" ? "asc" : s === "asc" ? "desc" : "scan"));

  // Fill the height left below the panels so the list and value viewer use the
  // whole viewport (matching the console's result table) instead of stopping
  // short. Re-measures under whatever chrome (e.g. the health banner) sits above.
  const { ref: gridRef, maxHeight } = useRemainingHeight<HTMLDivElement>();

  if (dbState.status === "loading") {
    return <Skeleton className="h-48 w-full" />;
  }
  if (dbState.status === "error" || dbState.data.error) {
    const message =
      dbState.status === "error" ? dbState.error.message : dbState.data.error;
    return (
      <Alert variant="destructive">
        <AlertCircle />
        <div>
          <AlertTitle>Redis unreachable</AlertTitle>
          <AlertDescription className="font-mono text-xs">
            {message}
          </AlertDescription>
        </div>
      </Alert>
    );
  }

  const databases = Math.max(1, dbState.data.count);

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-2">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="justify-between gap-2"
              aria-label="Select database"
            >
              <span className="flex items-center gap-2">
                <Database className="size-4 text-muted-foreground" aria-hidden />
                db {db}
              </span>
              <ChevronsUpDown className="size-4 text-muted-foreground" aria-hidden />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent
            align="start"
            className="max-h-72 overflow-y-auto"
          >
            {Array.from({ length: databases }, (_, i) => (
              <DropdownMenuItem
                key={i}
                onSelect={() => {
                  setSelectedKey(null);
                  setDb(i);
                }}
              >
                db {i}
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>

        <form
          className="flex min-w-0 flex-1 items-center gap-2"
          onSubmit={(e) => {
            e.preventDefault();
            setSelectedKey(null);
            applyPattern();
          }}
        >
          <Input
            value={draftPattern}
            onChange={(e) => setDraftPattern(e.target.value)}
            placeholder="Match pattern, e.g. user:*"
            aria-label="Key match pattern"
            className="min-w-0 flex-1 font-mono sm:max-w-xs"
          />
          <Button type="submit" size="sm" variant="secondary">
            <Search aria-hidden /> Scan
          </Button>
        </form>
      </div>

      <div
        ref={gridRef}
        style={{ height: maxHeight ?? undefined }}
        // Below md the panels stack, so let the page flow (override the fixed
        // fill height); from md they sit side by side and fill the viewport.
        className="grid min-h-[24rem] gap-4 max-md:!h-auto md:grid-cols-[minmax(0,20rem)_minmax(0,1fr)]"
      >
        <KeyList
          list={list}
          keys={sortedKeys}
          sort={sort}
          onCycleSort={cycleSort}
          selectedKey={selectedKey}
          onSelect={setSelectedKey}
          onLoadMore={() => loadPage(list.cursor, true)}
        />
        <RedisValuePanel
          profile={profile}
          service={service}
          db={db}
          selectedKey={selectedKey}
        />
      </div>
    </div>
  );
}

/** Labels and icons for the three sort modes, cycled by one button. */
const SORT_META: Record<SortMode, { label: string; icon: typeof ListOrdered }> = {
  scan: { label: "Scan order", icon: ListOrdered },
  asc: { label: "A → Z", icon: ArrowDownAZ },
  desc: { label: "Z → A", icon: ArrowUpAZ },
};

/** The scrollable key list with type/TTL badges and a "load more" action. */
function KeyList({
  list,
  keys,
  sort,
  onCycleSort,
  selectedKey,
  onSelect,
  onLoadMore,
}: {
  list: ListState;
  keys: KeyEntry[];
  sort: SortMode;
  onCycleSort: () => void;
  selectedKey: string | null;
  onSelect: (key: string) => void;
  onLoadMore: () => void;
}) {
  const SortIcon = SORT_META[sort].icon;
  const hasMore = list.cursor !== 0;

  // Header summarising what was scanned, plus the sort toggle. SCAN is
  // cursor-based, so we can only count what has loaded — say so honestly.
  const header = (
    <div className="flex items-center justify-between gap-2">
      <span className="truncate text-xs text-muted-foreground">
        {list.status === "loading"
          ? "Scanning…"
          : `${keys.length} ${keys.length === 1 ? "key" : "keys"}${hasMore ? " loaded · more available" : ""}`}
      </span>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        className="h-7 shrink-0 gap-1.5 px-2 text-xs text-muted-foreground"
        onClick={onCycleSort}
        disabled={keys.length === 0}
        aria-label={`Sort: ${SORT_META[sort].label}`}
      >
        <SortIcon className="size-3.5" aria-hidden />
        {SORT_META[sort].label}
      </Button>
    </div>
  );

  if (list.status === "loading") {
    return (
      <div className="flex h-full min-h-0 flex-col gap-2">
        {header}
        <Skeleton className="min-h-0 flex-1" />
      </div>
    );
  }
  if (list.status === "error") {
    return (
      <Alert variant="destructive" className="h-full">
        <AlertCircle />
        <div>
          <AlertTitle>Scan failed</AlertTitle>
          <AlertDescription className="font-mono text-xs">
            {list.error}
          </AlertDescription>
        </div>
      </Alert>
    );
  }
  if (keys.length === 0) {
    return (
      <div className="flex h-full min-h-0 flex-col gap-2">
        {header}
        <div className="flex min-h-0 flex-1 items-center justify-center rounded-md border border-dashed border-border p-6 text-center">
          <p className="text-sm text-muted-foreground">No keys match.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-full min-h-0 flex-col gap-2">
      {header}
      <ul className="min-h-0 flex-1 divide-y divide-border overflow-auto rounded-md border border-border">
        {keys.map((entry) => (
          <li
            key={entry.name}
            className={cn(
              "group flex items-center gap-1 pr-1 hover:bg-muted",
              entry.name === selectedKey && "bg-muted",
            )}
          >
            <button
              type="button"
              onClick={() => onSelect(entry.name)}
              aria-current={entry.name === selectedKey}
              className="flex min-w-0 flex-1 items-center gap-2 px-3 py-1.5 text-left"
            >
              <span className="min-w-0 truncate font-mono text-sm">
                {entry.name}
              </span>
              <RedisTypeBadge type={entry.type} className="ml-auto shrink-0" />
            </button>
            <CopyButton
              value={entry.name}
              label="key"
              className="shrink-0 opacity-0 group-hover:opacity-100 focus-visible:opacity-100"
            />
          </li>
        ))}
      </ul>
      {hasMore && (
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="w-full"
          disabled={list.status === "more"}
          onClick={onLoadMore}
        >
          {list.status === "more" ? "Loading…" : "Load more"}
        </Button>
      )}
    </div>
  );
}
