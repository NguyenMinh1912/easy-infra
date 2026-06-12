import { useEffect, useState } from "react";
import { Loader2, Plus } from "lucide-react";

import { Button } from "@/components/ui/button";
import type { CatalogEntry } from "@/types/service";

interface AddServiceFormProps {
  /** Catalog services not yet defined by the project. */
  available: CatalogEntry[];
  busy: boolean;
  onAdd: (name: string) => void;
}

/**
 * Adds a service to the project. Lets the user pick from the services
 * easy-infra supports that aren't defined yet; it is added with its default
 * definition, which can then be edited in place.
 */
export function AddServiceForm({ available, busy, onAdd }: AddServiceFormProps) {
  const [selected, setSelected] = useState("");

  // Keep the selection valid as the available set changes (e.g. after adding).
  useEffect(() => {
    setSelected((current) =>
      available.some((entry) => entry.name === current)
        ? current
        : (available[0]?.name ?? ""),
    );
  }, [available]);

  if (available.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        All supported services are already defined.
      </p>
    );
  }

  return (
    <form
      className="flex items-center gap-2"
      onSubmit={(e) => {
        e.preventDefault();
        if (selected) onAdd(selected);
      }}
    >
      <select
        aria-label="Service to add"
        className="h-9 rounded-md border border-input bg-background px-3 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:opacity-50"
        value={selected}
        disabled={busy}
        onChange={(e) => setSelected(e.target.value)}
      >
        {available.map((entry) => (
          <option key={entry.name} value={entry.name}>
            {entry.name}
          </option>
        ))}
      </select>
      <Button type="submit" size="sm" disabled={busy || !selected}>
        {busy ? <Loader2 className="animate-spin" /> : <Plus />}
        Add service
      </Button>
    </form>
  );
}
