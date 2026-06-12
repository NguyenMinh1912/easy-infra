import { Plus, X } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type { ServiceConfig } from "@/types/service";

/** One editable key/value pair of a service definition. */
export interface DefinitionRow {
  key: string;
  value: string;
}

/** Expand a service definition into editable rows (values stringified). */
export function rowsFromConfig(definition: ServiceConfig): DefinitionRow[] {
  return Object.entries(definition).map(([key, value]) => ({
    key,
    value: String(value),
  }));
}

/** Collapse editable rows back into a definition, dropping blank keys. */
export function configFromRows(rows: DefinitionRow[]): ServiceConfig {
  const config: ServiceConfig = {};
  for (const { key, value } of rows) {
    const trimmed = key.trim();
    if (trimmed) {
      config[trimmed] = value;
    }
  }
  return config;
}

interface DefinitionEditorProps {
  rows: DefinitionRow[];
  onChange: (rows: DefinitionRow[]) => void;
  disabled?: boolean;
}

/**
 * Service-agnostic editor for a definition's key/value pairs. The project
 * definition is a free-form string map, so the editor stays generic rather than
 * special-casing per service (mirroring the backend's `service.Config`).
 */
export function DefinitionEditor({
  rows,
  onChange,
  disabled,
}: DefinitionEditorProps) {
  const update = (index: number, patch: Partial<DefinitionRow>) =>
    onChange(rows.map((row, i) => (i === index ? { ...row, ...patch } : row)));

  const remove = (index: number) =>
    onChange(rows.filter((_, i) => i !== index));

  const add = () => onChange([...rows, { key: "", value: "" }]);

  return (
    <div className="space-y-2">
      {rows.map((row, i) => (
        <div key={i} className="flex items-center gap-2">
          <Input
            aria-label="Setting name"
            placeholder="key"
            className="w-40"
            value={row.key}
            disabled={disabled}
            onChange={(e) => update(i, { key: e.target.value })}
          />
          <Input
            aria-label={`Value for ${row.key || "setting"}`}
            placeholder="value"
            value={row.value}
            disabled={disabled}
            onChange={(e) => update(i, { value: e.target.value })}
          />
          <Button
            type="button"
            variant="ghost"
            size="icon"
            aria-label={`Remove ${row.key || "setting"}`}
            disabled={disabled}
            onClick={() => remove(i)}
          >
            <X />
          </Button>
        </div>
      ))}
      <Button
        type="button"
        variant="outline"
        size="sm"
        disabled={disabled}
        onClick={add}
      >
        <Plus />
        Add setting
      </Button>
    </div>
  );
}
