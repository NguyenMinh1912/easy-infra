import { useState } from "react";
import { Check, Loader2, RotateCcw } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { cn } from "@/lib/utils";
import type { CatalogEntry, ServiceConfig, ServiceInstance } from "@/types/service";
import { metaFor } from "../catalog-meta";
import {
  DefinitionEditor,
  configFromRows,
  rowsFromConfig,
  type DefinitionRow,
} from "./DefinitionEditor";

/** What the dialog is doing: adding a new service, or editing an existing one. */
export type DialogState =
  | { mode: "add" }
  | { mode: "edit"; service: ServiceInstance };

interface ServiceDialogProps {
  dialog: DialogState;
  /** Catalog services not yet defined — the choices in add mode. */
  available: CatalogEntry[];
  /** Full catalog, used to pre-fill defaults and to reset-to-defaults. */
  catalog: CatalogEntry[];
  busy: boolean;
  onClose: () => void;
  /** Persist the definition for `name` (parent decides create vs. update). */
  onSubmit: (name: string, definition: ServiceConfig) => void;
}

/**
 * Modal for adding or editing a service definition. In add mode the user picks
 * a service and tweaks its default settings before it is created; in edit mode
 * the settings of an existing service are edited. Both share the same key/value
 * editor and a reset-to-defaults action. Mounting/unmounting from the parent
 * gives the form fresh state each time it opens.
 */
export function ServiceDialog({
  dialog,
  available,
  catalog,
  busy,
  onClose,
  onSubmit,
}: ServiceDialogProps) {
  const isAdd = dialog.mode === "add";

  const defaultFor = (name: string): ServiceConfig =>
    catalog.find((entry) => entry.name === name)?.defaultConfig ?? {};

  const [selected, setSelected] = useState(() =>
    isAdd ? (available[0]?.name ?? "") : dialog.service.name,
  );
  const [rows, setRows] = useState<DefinitionRow[]>(() =>
    isAdd
      ? rowsFromConfig(defaultFor(available[0]?.name ?? ""))
      : rowsFromConfig(dialog.service.config),
  );

  // Switching the chosen service in add mode re-seeds the settings editor with
  // that service's defaults.
  const pick = (name: string) => {
    setSelected(name);
    setRows(rowsFromConfig(defaultFor(name)));
  };

  const reset = () => setRows(rowsFromConfig(defaultFor(selected)));

  const submit = () => {
    if (!selected) return;
    // In add mode the service is created with its defaults; its settings are
    // tuned in the settings modal that opens next. Edit mode persists the rows.
    onSubmit(selected, isAdd ? defaultFor(selected) : configFromRows(rows));
  };

  const meta = metaFor(selected);
  const Icon = meta.icon;

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <span className="flex size-8 items-center justify-center rounded-lg bg-muted">
              <Icon className="size-4 text-muted-foreground" aria-hidden />
            </span>
            {isAdd ? "Add a service" : `Edit ${meta.label}`}
          </DialogTitle>
          <DialogDescription>
            {isAdd
              ? "Choose a service to add to this profile. You can adjust its settings next."
              : "Edit this service's settings for this profile."}
          </DialogDescription>
        </DialogHeader>

        {isAdd && (
          <fieldset className="space-y-2">
            <legend className="sr-only">Service to add</legend>
            {available.map((entry) => {
              const entryMeta = metaFor(entry.name);
              const EntryIcon = entryMeta.icon;
              const active = selected === entry.name;
              return (
                <label
                  key={entry.name}
                  className={cn(
                    "flex cursor-pointer items-center gap-3 rounded-lg border p-3 text-sm transition-colors focus-within:ring-2 focus-within:ring-ring",
                    active
                      ? "border-primary bg-accent"
                      : "hover:bg-accent/50",
                  )}
                >
                  <input
                    type="radio"
                    name="service"
                    value={entry.name}
                    checked={active}
                    disabled={busy}
                    onChange={() => pick(entry.name)}
                    className="sr-only"
                  />
                  <span className="flex size-8 items-center justify-center rounded-lg bg-muted">
                    <EntryIcon className="size-4 text-muted-foreground" aria-hidden />
                  </span>
                  <span className="flex-1">
                    <span className="font-medium">{entryMeta.label}</span>
                    <span className="block text-xs text-muted-foreground">
                      {entry.name} · {entryMeta.blurb}
                    </span>
                  </span>
                  {active && <Check className="size-4 text-primary" aria-hidden />}
                </label>
              );
            })}
          </fieldset>
        )}

        {!isAdd && (
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium">Settings</span>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                disabled={busy || !selected}
                onClick={reset}
              >
                <RotateCcw aria-hidden />
                Reset to defaults
              </Button>
            </div>
            <div className="max-h-[40vh] overflow-y-auto pr-1">
              <DefinitionEditor rows={rows} onChange={setRows} disabled={busy} />
            </div>
          </div>
        )}

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            disabled={busy}
            onClick={onClose}
          >
            Cancel
          </Button>
          <Button type="button" disabled={busy || !selected} onClick={submit}>
            {busy && <Loader2 className="animate-spin" aria-hidden />}
            {isAdd ? "Add service" : "Save"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
