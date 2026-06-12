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
import { Input } from "@/components/ui/input";
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

/** What the dialog hands back to the parent on submit. */
export interface ServiceDialogResult {
  /** In add mode, the chosen service type; in edit mode, the instance id. */
  target: string;
  /** The user-facing display name (may be empty to keep the default). */
  name: string;
  config: ServiceConfig;
}

interface ServiceDialogProps {
  dialog: DialogState;
  /** Catalog services offered as choices in add mode. */
  available: CatalogEntry[];
  /** Full catalog, used to pre-fill defaults and to reset-to-defaults. */
  catalog: CatalogEntry[];
  busy: boolean;
  onClose: () => void;
  /** Persist the result (parent decides create vs. update from the mode). */
  onSubmit: (result: ServiceDialogResult) => void;
}

/**
 * Modal for adding or editing a service instance. In add mode the user picks a
 * service type, names it, and tweaks its default settings before it is created;
 * in edit mode the name and settings of an existing instance are edited (its
 * type is fixed). A profile may hold several instances of the same type, so the
 * name is how the user tells them apart. Both modes share the same key/value
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

  const defaultFor = (type: string): ServiceConfig =>
    catalog.find((entry) => entry.name === type)?.defaultConfig ?? {};

  // The service type being configured: the chosen catalog entry in add mode, or
  // the existing instance's fixed type in edit mode.
  const [selectedType, setSelectedType] = useState(() =>
    isAdd ? (available[0]?.name ?? "") : dialog.service.type,
  );
  const [name, setName] = useState(() => (isAdd ? "" : dialog.service.name));
  const [rows, setRows] = useState<DefinitionRow[]>(() =>
    isAdd
      ? rowsFromConfig(defaultFor(available[0]?.name ?? ""))
      : rowsFromConfig(dialog.service.config),
  );

  // Switching the chosen service in add mode re-seeds the settings editor with
  // that service's defaults.
  const pick = (type: string) => {
    setSelectedType(type);
    setRows(rowsFromConfig(defaultFor(type)));
  };

  const reset = () => setRows(rowsFromConfig(defaultFor(selectedType)));

  const submit = () => {
    if (!selectedType) return;
    onSubmit({
      target: isAdd ? selectedType : dialog.service.id,
      name: name.trim(),
      config: configFromRows(rows),
    });
  };

  const meta = metaFor(selectedType);
  const Icon = meta.icon;

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <span className="flex size-8 items-center justify-center rounded-lg bg-muted">
              <Icon className="size-4 text-muted-foreground" aria-hidden />
            </span>
            {isAdd ? "Add a service" : `Edit ${dialog.service.name}`}
          </DialogTitle>
          <DialogDescription>
            {isAdd
              ? "Choose a service, name it, adjust its settings, then add it to this profile."
              : "Rename this service or edit its settings for this profile."}
          </DialogDescription>
        </DialogHeader>

        {isAdd && (
          <fieldset className="space-y-2">
            <legend className="sr-only">Service to add</legend>
            {available.map((entry) => {
              const entryMeta = metaFor(entry.name);
              const EntryIcon = entryMeta.icon;
              const active = selectedType === entry.name;
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

        <div className="space-y-1.5">
          <label htmlFor="service-name" className="text-sm font-medium">
            Name
          </label>
          <Input
            id="service-name"
            value={name}
            disabled={busy || !selectedType}
            placeholder={selectedType}
            onChange={(e) => setName(e.target.value)}
          />
          <p className="text-xs text-muted-foreground">
            A label to tell this service apart from others of the same type.
            Defaults to <span className="font-mono">{selectedType || "the service type"}</span>.
          </p>
        </div>

        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium">Settings</span>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              disabled={busy || !selectedType}
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

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            disabled={busy}
            onClick={onClose}
          >
            Cancel
          </Button>
          <Button type="button" disabled={busy || !selectedType} onClick={submit}>
            {busy && <Loader2 className="animate-spin" aria-hidden />}
            {isAdd ? "Add service" : "Save"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
