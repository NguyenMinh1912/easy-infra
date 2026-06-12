import { Check, ChevronsUpDown, Search, Table2 } from "lucide-react";
import { useMemo, useState } from "react";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { useRemainingHeight } from "@/hooks/useRemainingHeight";
import { useResizableWidth } from "@/hooks/useResizableWidth";
import { cn } from "@/lib/utils";
import type { TableInfo } from "@/types/console";

/** Sidebar width bounds (px); the default matches the former fixed `w-56`. */
const WIDTH = { initial: 224, min: 160, max: 480 } as const;

interface SchemaSidebarProps {
  /** Whether schema introspection is still in flight. */
  loading: boolean;
  /** True when introspection finished but no schema is available (error). */
  unavailable: boolean;
  /** Schemas the connection can see, used to populate the dropdown. */
  schemas: string[];
  /** The schema currently selected in the dropdown. */
  selected: string | null;
  onSelect: (schema: string) => void;
  /** Tables belonging to the selected schema. */
  tables: TableInfo[];
  /** Double-clicking a table opens a console previewing its rows. */
  onTableOpen?: (table: TableInfo) => void;
}

/**
 * Console sidebar: a dropdown to pick a schema (defaulting to the profile's
 * configured schema) and, below it, the tables in that schema. Purely a
 * navigational aid for browsing the namespace the statements run against.
 */
export function SchemaSidebar({
  loading,
  unavailable,
  schemas,
  selected,
  onSelect,
  tables,
  onTableOpen,
}: SchemaSidebarProps) {
  // Cap the table list to the height still visible below it so a long list
  // scrolls within the sidebar instead of growing the page past the viewport.
  const { ref, maxHeight } = useRemainingHeight<HTMLUListElement>();
  // Width is user-draggable (handle on the right edge) and remembered across
  // reloads.
  const { width, onResizeStart } = useResizableWidth({
    key: "console-schema-sidebar",
    ...WIDTH,
  });
  // Free-text filter over the table names, applied case-insensitively.
  const [filter, setFilter] = useState("");
  const query = filter.trim().toLowerCase();
  const filteredTables = useMemo(
    () =>
      query
        ? tables.filter((table) => table.name.toLowerCase().includes(query))
        : tables,
    [tables, query],
  );
  return (
    <aside
      style={{ width }}
      className="relative shrink-0 space-y-3 border-r border-border pr-4"
    >
      <div className="space-y-1.5">
        <p className="text-xs font-medium text-muted-foreground">Schema</p>
        {loading ? (
          <Skeleton className="h-9 w-full" />
        ) : (
          <DropdownMenu>
            <DropdownMenuTrigger
              disabled={unavailable || schemas.length === 0}
              className="flex h-9 w-full items-center justify-between rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
            >
              <span className="truncate">
                {selected ?? (unavailable ? "Unavailable" : "—")}
              </span>
              <ChevronsUpDown
                className="ml-2 size-4 shrink-0 text-muted-foreground"
                aria-hidden
              />
            </DropdownMenuTrigger>
            <DropdownMenuContent
              align="start"
              className="max-h-72 w-[var(--radix-dropdown-menu-trigger-width)] overflow-y-auto"
            >
              {schemas.map((schema) => (
                <DropdownMenuItem
                  key={schema}
                  onSelect={() => onSelect(schema)}
                  className="justify-between"
                >
                  <span className="truncate">{schema}</span>
                  {schema === selected && (
                    <Check className="size-4 shrink-0" aria-hidden />
                  )}
                </DropdownMenuItem>
              ))}
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>

      <div className="space-y-1">
        <p className="text-xs font-medium text-muted-foreground">
          Tables{!loading && !unavailable && ` (${filteredTables.length})`}
        </p>
        {loading ? (
          <div className="space-y-1.5">
            <Skeleton className="h-6 w-full" />
            <Skeleton className="h-6 w-5/6" />
            <Skeleton className="h-6 w-2/3" />
          </div>
        ) : unavailable ? (
          <p className="text-xs text-muted-foreground">
            Schema unavailable — the database could not be introspected.
          </p>
        ) : tables.length === 0 ? (
          <p className="text-xs text-muted-foreground">No tables.</p>
        ) : (
          <>
            <div className="relative">
              <Search
                className="pointer-events-none absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
                aria-hidden
              />
              <Input
                type="search"
                value={filter}
                onChange={(e) => setFilter(e.target.value)}
                placeholder="Filter tables…"
                aria-label="Filter tables by name"
                className="h-8 pl-8 text-sm"
              />
            </div>
            {filteredTables.length === 0 ? (
              <p className="text-xs text-muted-foreground">
                No tables match “{filter.trim()}”.
              </p>
            ) : (
              <ul
                ref={ref}
                style={{ maxHeight: maxHeight ?? undefined }}
                className="space-y-0.5 overflow-y-auto"
              >
                {filteredTables.map((table) => (
                  <li
                    key={table.name}
                    title={`${table.name} · ${table.columns.length} columns${
                      onTableOpen ? " · double-click to preview rows" : ""
                    }`}
                    onDoubleClick={() => onTableOpen?.(table)}
                    className={cn(
                      "flex items-center gap-2 rounded-sm px-2 py-1 text-sm",
                      "text-foreground",
                      onTableOpen &&
                        "cursor-pointer select-none hover:bg-muted",
                    )}
                  >
                    <Table2
                      className="size-4 shrink-0 text-muted-foreground"
                      aria-hidden
                    />
                    <span className="truncate">{table.name}</span>
                  </li>
                ))}
              </ul>
            )}
          </>
        )}
      </div>

      <div
        role="separator"
        aria-orientation="vertical"
        aria-label="Resize table list"
        onPointerDown={onResizeStart}
        className="absolute inset-y-0 -right-2 w-4 cursor-col-resize touch-none after:absolute after:inset-y-0 after:left-1/2 after:w-px after:-translate-x-1/2 after:bg-transparent hover:after:bg-border"
      />
    </aside>
  );
}
