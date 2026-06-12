import { useCallback, useState } from "react";
import { toast } from "sonner";

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { createService, deleteService, updateService } from "@/services/api";
import type { ServiceConfig } from "@/types/service";
import type { ServicesData } from "../hooks/useServices";
import { AddServiceForm } from "./AddServiceForm";
import { ServiceCard } from "./ServiceCard";

interface ServicesManagerProps {
  data: ServicesData;
  reload: () => void;
  /** Service to scroll to and highlight, deep-linked from the sidebar. */
  focusService?: string;
}

/**
 * Presentation + mutation wiring for the services screen. It composes the add
 * form and the per-service cards, runs each create/update/delete through a
 * single helper that toasts the outcome and refreshes the list on success.
 */
export function ServicesManager({
  data,
  reload,
  focusService,
}: ServicesManagerProps) {
  const [busy, setBusy] = useState(false);

  const run = useCallback(
    async (action: () => Promise<unknown>, messages: { success: string; error: string }) => {
      setBusy(true);
      try {
        await action();
        toast.success(messages.success);
        reload();
      } catch (cause) {
        toast.error(messages.error, {
          description: cause instanceof Error ? cause.message : String(cause),
        });
      } finally {
        setBusy(false);
      }
    },
    [reload],
  );

  const defined = new Set(data.services.map((s) => s.name));
  const available = data.catalog.filter((entry) => !defined.has(entry.name));

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="text-sm font-medium uppercase tracking-wide text-muted-foreground">
            Add a service
          </CardTitle>
        </CardHeader>
        <CardContent>
          <AddServiceForm
            available={available}
            busy={busy}
            onAdd={(name) =>
              run(() => createService(name), {
                success: `Service "${name}" added`,
                error: `Could not add "${name}"`,
              })
            }
          />
        </CardContent>
      </Card>

      {data.services.length === 0 ? (
        <p className="text-sm text-muted-foreground">
          No services defined yet. Add one above to get started.
        </p>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2">
          {data.services.map((service) => (
            <ServiceCard
              key={service.name}
              service={service}
              busy={busy}
              highlighted={service.name === focusService}
              onSave={(name, definition: ServiceConfig) =>
                run(() => updateService(name, definition), {
                  success: `Service "${name}" updated`,
                  error: `Could not update "${name}"`,
                })
              }
              onRemove={(name) =>
                run(() => deleteService(name), {
                  success: `Service "${name}" removed`,
                  error: `Could not remove "${name}"`,
                })
              }
            />
          ))}
        </div>
      )}
    </div>
  );
}
