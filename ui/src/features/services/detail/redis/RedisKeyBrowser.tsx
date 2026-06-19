import { AlertCircle, ChevronsUpDown, Database, Search } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { useAsync } from "@/hooks/useAsync";
import { getDatabases, listKeys } from "@/services/api";
import type { KeyEntry } from "@/types/redis";

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

      <div className="grid gap-4 md:grid-cols-[minmax(0,18rem)_minmax(0,1fr)]">
        <KeyList
          list={list}
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

/** The scrollable key list with type/TTL badges and a "load more" action. */
function KeyList({
  list,
  selectedKey,
  onSelect,
  onLoadMore,
}: {
  list: ListState;
  selectedKey: string | null;
  onSelect: (key: string) => void;
  onLoadMore: () => void;
}) {
  if (list.status === "loading") {
    return <Skeleton className="h-48 w-full" />;
  }
  if (list.status === "error") {
    return (
      <Alert variant="destructive">
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
  if (list.keys.length === 0) {
    return (
      <div className="rounded-md border border-dashed border-border p-6 text-center">
        <p className="text-sm text-muted-foreground">No keys match.</p>
      </div>
    );
  }

  return (
    <div className="space-y-2">
      <ul className="max-h-96 divide-y divide-border overflow-auto rounded-md border border-border">
        {list.keys.map((entry) => (
          <li key={entry.name}>
            <button
              type="button"
              onClick={() => onSelect(entry.name)}
              aria-current={entry.name === selectedKey}
              className="flex w-full items-center justify-between gap-2 px-3 py-2 text-left hover:bg-muted aria-[current=true]:bg-muted"
            >
              <span className="min-w-0 truncate font-mono text-sm">
                {entry.name}
              </span>
              <Badge variant="secondary" className="shrink-0">
                {entry.type}
              </Badge>
            </button>
          </li>
        ))}
      </ul>
      {list.cursor !== 0 && (
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
