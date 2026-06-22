import { useMemo, useState } from "react";
import { Plus, Search } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import type { TableInfo } from "@/types/console";

interface AddTableControlProps {
  /** Tables the connection exposes, used to populate the picker. */
  tables: TableInfo[];
  /** Add the chosen table to the canvas as a starting node. */
  onAdd: (schema: string, table: string) => void;
  disabled?: boolean;
}

/**
 * A searchable dropdown of the schema's tables; choosing one drops it onto the
 * canvas as a starting node to expand from. The filter input stops its keydown
 * from bubbling so the menu's typeahead doesn't hijack what's typed.
 */
export function AddTableControl({
  tables,
  onAdd,
  disabled,
}: AddTableControlProps) {
  const [filter, setFilter] = useState("");
  const query = filter.trim().toLowerCase();
  const filtered = useMemo(
    () =>
      query === ""
        ? tables
        : tables.filter((t) =>
            `${t.schema}.${t.name}`.toLowerCase().includes(query),
          ),
    [query, tables],
  );

  return (
    <DropdownMenu onOpenChange={(open) => !open && setFilter("")}>
      <DropdownMenuTrigger asChild>
        <Button size="sm" disabled={disabled}>
          <Plus aria-hidden /> Add table
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-72 p-0">
        <div className="relative border-b border-border p-2">
          <Search className="absolute left-4 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
          <Input
            autoFocus
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            onKeyDown={(e) => e.stopPropagation()}
            placeholder="Filter tables…"
            className="h-8 pl-7 text-xs"
          />
        </div>
        <ul className="max-h-72 overflow-auto p-1">
          {filtered.length === 0 ? (
            <li className="px-2 py-3 text-center text-xs text-muted-foreground">
              No tables match
            </li>
          ) : (
            filtered.map((t) => (
              <DropdownMenuItem
                key={`${t.schema}.${t.name}`}
                onSelect={() => onAdd(t.schema, t.name)}
                className="font-mono text-xs"
              >
                <span className="truncate">
                  <span className="text-muted-foreground">{t.schema}.</span>
                  {t.name}
                </span>
              </DropdownMenuItem>
            ))
          )}
        </ul>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
