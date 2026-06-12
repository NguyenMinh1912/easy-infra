import { useState } from "react";
import { Pencil, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import type { ServiceConfig, ServiceDefinition } from "@/types/service";
import { DefinitionEditor, type DefinitionRow } from "./DefinitionEditor";

interface ServiceCardProps {
  service: ServiceDefinition;
  busy: boolean;
  onSave: (name: string, definition: ServiceConfig) => void;
  onRemove: (name: string) => void;
}

/**
 * One service definition: shows its settings, with inline editing and removal.
 * Owns only its local edit-mode/draft state; the actual mutations are delegated
 * to the parent via `onSave`/`onRemove`.
 */
export function ServiceCard({
  service,
  busy,
  onSave,
  onRemove,
}: ServiceCardProps) {
  const [editing, setEditing] = useState(false);
  const [rows, setRows] = useState<DefinitionRow[]>([]);

  const startEditing = () => {
    setRows(toRows(service.definition));
    setEditing(true);
  };

  const save = () => {
    onSave(service.name, toConfig(rows));
    setEditing(false);
  };

  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0">
        <CardTitle className="font-mono text-base">{service.name}</CardTitle>
        {!editing && (
          <div className="flex gap-1">
            <Button
              variant="ghost"
              size="icon"
              aria-label={`Edit ${service.name}`}
              disabled={busy}
              onClick={startEditing}
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
              description="This drops the service and its config from every profile. This action cannot be undone."
              confirmLabel="Remove"
              variant="destructive"
              onConfirm={() => onRemove(service.name)}
            />
          </div>
        )}
      </CardHeader>
      <CardContent>
        {editing ? (
          <div className="space-y-4">
            <DefinitionEditor rows={rows} onChange={setRows} disabled={busy} />
            <div className="flex gap-2">
              <Button size="sm" disabled={busy} onClick={save}>
                Save
              </Button>
              <Button
                size="sm"
                variant="outline"
                disabled={busy}
                onClick={() => setEditing(false)}
              >
                Cancel
              </Button>
            </div>
          </div>
        ) : (
          <Definitions definition={service.definition} />
        )}
      </CardContent>
    </Card>
  );
}

/** Read-only view of a definition's settings. */
function Definitions({ definition }: { definition: ServiceConfig }) {
  const entries = Object.entries(definition);
  if (entries.length === 0) {
    return <p className="text-sm text-muted-foreground">No settings.</p>;
  }
  return (
    <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
      {entries.map(([key, value]) => (
        <div key={key} className="contents">
          <dt className="text-muted-foreground">{key}</dt>
          <dd className="font-mono">{String(value)}</dd>
        </div>
      ))}
    </dl>
  );
}

function toRows(definition: ServiceConfig): DefinitionRow[] {
  return Object.entries(definition).map(([key, value]) => ({
    key,
    value: String(value),
  }));
}

function toConfig(rows: DefinitionRow[]): ServiceConfig {
  const config: ServiceConfig = {};
  for (const { key, value } of rows) {
    const trimmed = key.trim();
    if (trimmed) {
      config[trimmed] = value;
    }
  }
  return config;
}
