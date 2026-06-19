import { Check, ChevronsUpDown, Globe, Search } from "lucide-react";
import { useEffect, useId, useMemo, useRef, useState } from "react";

import { cn } from "@/lib/utils";

import { AWS_REGIONS, formatRegion } from "./regions";

interface RegionSelectProps {
  /** Currently selected region id. */
  value: string;
  /** Called with the chosen region id. */
  onChange: (region: string) => void;
}

/**
 * Searchable AWS region picker for the overview toolbar. A plain select is
 * unwieldy past ~20 regions, so this is a filterable combobox: a button trigger
 * opens a panel with a search box and a keyboard-navigable listbox (arrows to
 * move, Enter to pick, Esc to close). It matches the dark theme and is fully
 * labelled for assistive tech; the parent announces the change via aria-live.
 */
export function RegionSelect({ value, onChange }: RegionSelectProps) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [active, setActive] = useState(0);
  const rootRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const activeRef = useRef<HTMLLIElement>(null);
  const listId = useId();

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return AWS_REGIONS;
    return AWS_REGIONS.filter(
      (r) => r.id.includes(q) || r.label.toLowerCase().includes(q),
    );
  }, [query]);

  // Close when focus or a click lands outside the widget.
  useEffect(() => {
    if (!open) return;
    const onDown = (e: MouseEvent) => {
      if (rootRef.current && !rootRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", onDown);
    return () => document.removeEventListener("mousedown", onDown);
  }, [open]);

  // On open, focus the search box and reset the query/highlight.
  useEffect(() => {
    if (open) {
      setQuery("");
      setActive(0);
      inputRef.current?.focus();
    }
  }, [open]);

  // Keep the highlight on the first match as the query narrows.
  useEffect(() => setActive(0), [query]);

  // Scroll the highlighted option into view as the user arrows through.
  useEffect(() => {
    activeRef.current?.scrollIntoView({ block: "nearest" });
  }, [active, open]);

  const choose = (id: string) => {
    onChange(id);
    setOpen(false);
  };

  const onKeyDown = (e: React.KeyboardEvent) => {
    switch (e.key) {
      case "ArrowDown":
        e.preventDefault();
        setActive((a) => Math.min(a + 1, filtered.length - 1));
        break;
      case "ArrowUp":
        e.preventDefault();
        setActive((a) => Math.max(a - 1, 0));
        break;
      case "Enter": {
        e.preventDefault();
        const sel = filtered[active];
        if (sel) choose(sel.id);
        break;
      }
      case "Escape":
        e.preventDefault();
        setOpen(false);
        break;
    }
  };

  return (
    <div className="relative" ref={rootRef}>
      <button
        type="button"
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-label={`AWS region, currently ${formatRegion(value)}`}
        onClick={() => setOpen((o) => !o)}
        className="flex h-9 items-center gap-2 rounded-md border border-input bg-background px-3 text-sm transition-colors hover:bg-accent focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none"
      >
        <Globe className="size-4 text-muted-foreground" aria-hidden />
        <span className="font-medium">{value}</span>
        <ChevronsUpDown className="size-4 text-muted-foreground" aria-hidden />
      </button>

      {open && (
        <div className="absolute right-0 z-50 mt-1 w-72 overflow-hidden rounded-md border border-border bg-popover text-popover-foreground shadow-md">
          <div className="flex items-center gap-2 border-b border-border px-3">
            <Search className="size-4 shrink-0 text-muted-foreground" aria-hidden />
            <input
              ref={inputRef}
              type="text"
              role="combobox"
              aria-controls={listId}
              aria-expanded={open}
              aria-autocomplete="list"
              aria-label="Search regions"
              placeholder="Search regions…"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={onKeyDown}
              className="h-9 w-full bg-transparent py-1 text-sm outline-none placeholder:text-muted-foreground"
            />
          </div>
          <ul
            id={listId}
            role="listbox"
            aria-label="AWS regions"
            className="max-h-64 overflow-y-auto p-1"
          >
            {filtered.length === 0 ? (
              <li className="px-2 py-6 text-center text-sm text-muted-foreground">
                No matching regions
              </li>
            ) : (
              filtered.map((r, i) => {
                const selected = r.id === value;
                return (
                  <li
                    key={r.id}
                    ref={i === active ? activeRef : undefined}
                    role="option"
                    aria-selected={selected}
                    onMouseEnter={() => setActive(i)}
                    onClick={() => choose(r.id)}
                    className={cn(
                      "flex cursor-pointer items-center gap-2 rounded-sm px-2 py-1.5 text-sm",
                      i === active && "bg-accent text-accent-foreground",
                    )}
                  >
                    <Check
                      className={cn(
                        "size-4 shrink-0",
                        selected ? "opacity-100" : "opacity-0",
                      )}
                      aria-hidden
                    />
                    <span className="flex-1 truncate">{r.label}</span>
                    <span className="font-mono text-xs text-muted-foreground">
                      {r.id}
                    </span>
                  </li>
                );
              })
            )}
          </ul>
        </div>
      )}
    </div>
  );
}
