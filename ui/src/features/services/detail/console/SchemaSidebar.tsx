import { Check, ChevronsUpDown, Table2 } from "lucide-react";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import type { TableInfo } from "@/types/console";

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
}: SchemaSidebarProps) {
  return (
    <aside className="w-56 shrink-0 space-y-3 border-r border-border pr-4">
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
          Tables{!loading && !unavailable && ` (${tables.length})`}
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
          <ul className="space-y-0.5">
            {tables.map((table) => (
              <li
                key={table.name}
                title={`${table.name} · ${table.columns.length} columns`}
                className={cn(
                  "flex items-center gap-2 rounded-sm px-2 py-1 text-sm",
                  "text-foreground",
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
      </div>
    </aside>
  );
}
