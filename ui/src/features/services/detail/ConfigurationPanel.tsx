import { useState } from "react";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { updateService } from "@/services/api";
import type { ServiceDefinition } from "@/types/service";

import {
  DefinitionEditor,
  configFromRows,
  rowsFromConfig,
  type DefinitionRow,
} from "../components/DefinitionEditor";

interface ConfigurationPanelProps {
  service: ServiceDefinition;
  /** Refresh the parent's data after a successful save. */
  onSaved: () => void;
}

/**
 * Editable project-level definition for one service. Reuses the
 * service-agnostic {@link DefinitionEditor}, persists via
 * PUT /api/services/{name}, and asks the parent to reload on success. Holds the
 * draft locally so Reset restores the last saved values.
 */
export function ConfigurationPanel({
  service,
  onSaved,
}: ConfigurationPanelProps) {
  const [rows, setRows] = useState<DefinitionRow[]>(() =>
    rowsFromConfig(service.definition),
  );
  const [saving, setSaving] = useState(false);

  const save = async () => {
    setSaving(true);
    try {
      await updateService(service.name, configFromRows(rows));
      toast.success(`Service "${service.name}" updated`);
      onSaved();
    } catch (cause) {
      toast.error(`Could not update "${service.name}"`, {
        description: cause instanceof Error ? cause.message : String(cause),
      });
    } finally {
      setSaving(false);
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Definition</CardTitle>
        <CardDescription>
          Project-level settings stored in{" "}
          <code className="font-mono">easy-infra.yml</code>. Per-environment
          connection details live in each profile.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <DefinitionEditor rows={rows} onChange={setRows} disabled={saving} />
        <div className="flex gap-2">
          <Button disabled={saving} onClick={save}>
            {saving && <Loader2 className="animate-spin" aria-hidden />}
            Save changes
          </Button>
          <Button
            variant="outline"
            disabled={saving}
            onClick={() => setRows(rowsFromConfig(service.definition))}
          >
            Reset
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
