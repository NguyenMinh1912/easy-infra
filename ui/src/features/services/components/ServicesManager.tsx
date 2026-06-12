import { useCallback, useState } from "react";
import { Boxes, Plus } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { createService, deleteService, updateService } from "@/services/api";
import type { ServiceConfig } from "@/types/service";
import type { ServicesData } from "../hooks/useServices";
import { ServiceDialog, type DialogState } from "./ServiceDialog";
import { ServicesTable } from "./ServicesTable";

interface ServicesManagerProps {
  data: ServicesData;
  reload: () => void;
  /** Service to scroll to and highlight, deep-linked from the sidebar. */
  focusService?: string;
}

/**
 * Presentation + mutation wiring for the services screen. Renders a toolbar, the
 * services table (or an empty state), and the add/edit dialog. Each create /
 * update / delete runs through a single helper that toasts the outcome and
 * refreshes the list on success.
 */
export function ServicesManager({
  data,
  reload,
  focusService,
}: ServicesManagerProps) {
  const [busy, setBusy] = useState(false);
  const [dialog, setDialog] = useState<DialogState | null>(null);

  const run = useCallback(
    async (
      action: () => Promise<unknown>,
      messages: { success: string; error: string },
    ): Promise<boolean> => {
      setBusy(true);
      try {
        await action();
        toast.success(messages.success);
        reload();
        return true;
      } catch (cause) {
        toast.error(messages.error, {
          description: cause instanceof Error ? cause.message : String(cause),
        });
        return false;
      } finally {
        setBusy(false);
      }
    },
    [reload],
  );

  const defined = new Set(data.services.map((s) => s.name));
  const available = data.catalog.filter((entry) => !defined.has(entry.name));

  // Create with defaults, then persist any edits the user made in the dialog.
  const add = (name: string, definition: ServiceConfig) => {
    const defaults =
      data.catalog.find((entry) => entry.name === name)?.defaultDefinition ?? {};
    const edited = !sameConfig(definition, defaults);
    return run(
      async () => {
        await createService(name);
        if (edited) await updateService(name, definition);
      },
      { success: `Service "${name}" added`, error: `Could not add "${name}"` },
    );
  };

  const submit = async (name: string, definition: ServiceConfig) => {
    if (!dialog) return;
    const ok =
      dialog.mode === "add"
        ? await add(name, definition)
        : await run(() => updateService(name, definition), {
            success: `Service "${name}" updated`,
            error: `Could not update "${name}"`,
          });
    if (ok) setDialog(null);
  };

  const remove = (name: string) =>
    run(() => deleteService(name), {
      success: `Service "${name}" removed`,
      error: `Could not remove "${name}"`,
    });

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between gap-3">
        <p className="text-sm text-muted-foreground">
          {data.services.length} of {data.catalog.length} defined
        </p>
        <Button
          size="sm"
          disabled={busy || available.length === 0}
          title={
            available.length === 0
              ? "All supported services are already defined"
              : undefined
          }
          onClick={() => setDialog({ mode: "add" })}
        >
          <Plus />
          Add service
        </Button>
      </div>

      {data.services.length === 0 ? (
        <Card className="flex flex-col items-center gap-3 p-10 text-center">
          <span className="flex size-10 items-center justify-center rounded-lg bg-muted">
            <Boxes className="size-5 text-muted-foreground" aria-hidden />
          </span>
          <div className="space-y-1">
            <p className="font-medium">No services defined yet</p>
            <p className="text-sm text-muted-foreground">
              Add your first service to get started.
            </p>
          </div>
          <Button
            size="sm"
            disabled={busy || available.length === 0}
            onClick={() => setDialog({ mode: "add" })}
          >
            <Plus />
            Add service
          </Button>
        </Card>
      ) : (
        <Card>
          <ServicesTable
            services={data.services}
            busy={busy}
            focusService={focusService}
            onEdit={(service) => setDialog({ mode: "edit", service })}
            onRemove={remove}
          />
        </Card>
      )}

      {dialog && (
        <ServiceDialog
          dialog={dialog}
          available={available}
          catalog={data.catalog}
          busy={busy}
          onClose={() => setDialog(null)}
          onSubmit={submit}
        />
      )}
    </div>
  );
}

/** Whether two definitions are equal once values are compared as strings. */
function sameConfig(a: ServiceConfig, b: ServiceConfig): boolean {
  const keys = Object.keys(a);
  if (keys.length !== Object.keys(b).length) return false;
  return keys.every((key) => key in b && String(a[key]) === String(b[key]));
}
