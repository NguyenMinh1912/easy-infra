import { useEffect, useRef } from "react";
import { Pencil, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { cn } from "@/lib/utils";
import type { ServiceConfig, ServiceInstance } from "@/types/service";
import { metaFor } from "../catalog-meta";

interface ServicesTableProps {
  services: ServiceInstance[];
  busy: boolean;
  /** Service to scroll to and highlight, deep-linked from the sidebar. */
  focusService?: string;
  onEdit: (service: ServiceInstance) => void;
  onRemove: (name: string) => void;
}

/** How many setting pairs to show inline before relying on the count. */
const PREVIEW_LIMIT = 3;

/**
 * The defined services, one row each: identity (icon + friendly label + raw
 * name), a compact settings preview with a count, and edit/remove actions.
 * Holds no mutation logic — edit and remove are delegated to the parent.
 */
export function ServicesTable({
  services,
  busy,
  focusService,
  onEdit,
  onRemove,
}: ServicesTableProps) {
  const focusRef = useRef<HTMLTableRowElement>(null);

  // Deep-linked from the sidebar: bring the highlighted row into view.
  useEffect(() => {
    if (focusService) {
      focusRef.current?.scrollIntoView({ behavior: "smooth", block: "center" });
    }
  }, [focusService]);

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Service</TableHead>
          <TableHead>Settings</TableHead>
          <TableHead className="text-right">Actions</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {services.map((service) => {
          const meta = metaFor(service.name);
          const Icon = meta.icon;
          const highlighted = service.name === focusService;
          return (
            <TableRow
              key={service.name}
              ref={highlighted ? focusRef : undefined}
              className={cn("scroll-mt-6", highlighted && "bg-accent")}
            >
              <TableCell>
                <div className="flex items-center gap-3">
                  <span className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-muted">
                    <Icon className="size-4 text-muted-foreground" aria-hidden />
                  </span>
                  <span>
                    <span className="font-medium">{meta.label}</span>
                    <span className="block font-mono text-xs text-muted-foreground">
                      {service.name}
                    </span>
                  </span>
                </div>
              </TableCell>
              <TableCell>
                <span className="block max-w-md truncate font-mono text-xs text-muted-foreground">
                  {preview(service.config)}
                </span>
                <span className="mt-0.5 block text-xs text-muted-foreground">
                  {count(service.config)}
                </span>
              </TableCell>
              <TableCell>
                <div className="flex items-center justify-end gap-1">
                  <Button
                    variant="ghost"
                    size="icon"
                    aria-label={`Edit ${service.name}`}
                    disabled={busy}
                    onClick={() => onEdit(service)}
                  >
                    <Pencil />
                  </Button>
                  <ConfirmDialog
                    trigger={
                      <Button
                        variant="ghost"
                        size="icon"
                        aria-label={`Remove ${service.name}`}
                        disabled={busy}
                      >
                        <Trash2 />
                      </Button>
                    }
                    title={`Remove "${service.name}"?`}
                    description="This removes the service and its config from this profile. This action cannot be undone."
                    confirmLabel="Remove"
                    variant="destructive"
                    onConfirm={() => onRemove(service.name)}
                  />
                </div>
              </TableCell>
            </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );
}

/** A one-line preview of the first few settings, e.g. "version 16 · port 5432". */
function preview(definition: ServiceConfig): string {
  const entries = Object.entries(definition);
  if (entries.length === 0) return "No settings";
  const shown = entries
    .slice(0, PREVIEW_LIMIT)
    .map(([key, value]) => `${key} ${String(value)}`)
    .join("  ·  ");
  return entries.length > PREVIEW_LIMIT ? `${shown}  ·  …` : shown;
}

/** "N settings" label, pluralized. */
function count(definition: ServiceConfig): string {
  const n = Object.keys(definition).length;
  return `${n} setting${n === 1 ? "" : "s"}`;
}
