import { Database } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

import { DefinitionSummary } from "./DefinitionSummary";
import type { OverviewProps } from "./types";

/**
 * Postgres-specific overview. Surfaces the image version prominently and
 * explains the split between the project-level definition (this screen) and the
 * per-profile connection settings, then lists the raw definition. The first
 * tailored service view embedded in {@link ServiceDetailLayout}.
 */
export function PostgresOverview({ service }: OverviewProps) {
  const version = String(service.definition.version ?? "—");

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <div className="flex items-center gap-3">
            <span className="flex size-10 items-center justify-center rounded-lg bg-muted">
              <Database className="size-5 text-muted-foreground" aria-hidden />
            </span>
            <div className="flex-1">
              <CardTitle className="text-base">PostgreSQL</CardTitle>
              <p className="text-sm text-muted-foreground">
                Relational database
              </p>
            </div>
            <Badge variant="secondary">v{version}</Badge>
          </div>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">
            This is the project-level definition (image version) tracked in{" "}
            <code className="font-mono text-foreground">easy-infra.yml</code>.
            Per-environment connection details — host, port, credentials,
            database — are configured per profile under{" "}
            <span className="font-medium text-foreground">
              Profiles → Settings
            </span>
            .
          </p>
        </CardContent>
      </Card>
      <DefinitionSummary definition={service.definition} />
    </div>
  );
}
