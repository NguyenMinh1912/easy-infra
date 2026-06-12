import { useCallback, useState } from "react";
import { AlertCircle } from "lucide-react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
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
}

/**
 * Presentation + mutation wiring for the services screen. It composes the add
 * form and the per-service cards, runs each create/update/delete through a
 * single helper that surfaces errors and refreshes the list on success.
 */
export function ServicesManager({ data, reload }: ServicesManagerProps) {
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const run = useCallback(
    async (action: () => Promise<unknown>) => {
      setBusy(true);
      setError(null);
      try {
        await action();
        reload();
      } catch (cause) {
        setError(cause instanceof Error ? cause.message : String(cause));
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
            onAdd={(name) => run(() => createService(name))}
          />
        </CardContent>
      </Card>

      {error && (
        <Alert variant="destructive">
          <AlertCircle />
          <div>
            <AlertTitle>Action failed</AlertTitle>
            <AlertDescription>{error}</AlertDescription>
          </div>
        </Alert>
      )}

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
              onSave={(name, definition: ServiceConfig) =>
                run(() => updateService(name, definition))
              }
              onRemove={(name) => run(() => deleteService(name))}
            />
          ))}
        </div>
      )}
    </div>
  );
}
