import { Plus, X } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";

/** One editable key/value pair of a service's environment config. */
export interface ConfigRow {
  key: string;
  value: string;
}

interface ProfileServiceConfigCardProps {
  /** Service this card configures (e.g. "postgres"). */
  name: string;
  rows: ConfigRow[];
  onChange: (rows: ConfigRow[]) => void;
  disabled?: boolean;
}

/**
 * Editable environment config for one service within a profile. The profile
 * env block is a free-form string map (host, port, credentials, …), so the
 * editor stays generic rather than special-casing per service, mirroring the
 * backend's `service.Config`. Owns no state — the parent holds the draft.
 */
export function ProfileServiceConfigCard({
  name,
  rows,
  onChange,
  disabled,
}: ProfileServiceConfigCardProps) {
  const update = (index: number, patch: Partial<ConfigRow>) =>
    onChange(rows.map((row, i) => (i === index ? { ...row, ...patch } : row)));

  const remove = (index: number) =>
    onChange(rows.filter((_, i) => i !== index));

  const add = () => onChange([...rows, { key: "", value: "" }]);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="font-mono text-base">{name}</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="space-y-2">
          {rows.map((row, i) => (
            <div key={i} className="flex items-center gap-2">
              <Input
                aria-label={`Setting name for ${name}`}
                placeholder="key"
                className="w-40"
                value={row.key}
                disabled={disabled}
                onChange={(e) => update(i, { key: e.target.value })}
              />
              <Input
                aria-label={`Value for ${row.key || "setting"} on ${name}`}
                placeholder="value"
                value={row.value}
                disabled={disabled}
                onChange={(e) => update(i, { value: e.target.value })}
              />
              <Button
                type="button"
                variant="ghost"
                size="icon"
                aria-label={`Remove ${row.key || "setting"} from ${name}`}
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
      </CardContent>
    </Card>
  );
}
