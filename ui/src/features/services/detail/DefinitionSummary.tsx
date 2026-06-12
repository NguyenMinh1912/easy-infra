import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { ServiceConfig } from "@/types/service";

interface DefinitionSummaryProps {
  definition: ServiceConfig;
}

/**
 * Read-only view of a service's project-level definition as key/value pairs.
 * Service-agnostic — the definition is a free-form string map, so every
 * overview can drop this in to render the current settings without editing.
 */
export function DefinitionSummary({ definition }: DefinitionSummaryProps) {
  const entries = Object.entries(definition);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Definition</CardTitle>
      </CardHeader>
      <CardContent>
        {entries.length === 0 ? (
          <p className="text-sm text-muted-foreground">No settings defined.</p>
        ) : (
          <dl className="grid gap-x-8 gap-y-2 sm:grid-cols-2">
            {entries.map(([key, value]) => (
              <div
                key={key}
                className="flex items-center justify-between gap-4 border-b border-border/50 pb-1.5"
              >
                <dt className="font-mono text-xs text-muted-foreground">
                  {key}
                </dt>
                <dd className="font-mono text-sm">{String(value)}</dd>
              </div>
            ))}
          </dl>
        )}
      </CardContent>
    </Card>
  );
}
