import { Card, CardHeader, CardTitle } from "@/components/ui/card";

import { metaFor } from "../catalog-meta";
import { DefinitionSummary } from "./DefinitionSummary";
import type { OverviewProps } from "./types";

/**
 * Generic service overview: identity (icon + label + blurb) and a read-only
 * summary of the definition. The fallback used for any service without a
 * tailored overview, so a new backend service renders without a code change.
 */
export function OverviewPanel({ service }: OverviewProps) {
  const meta = metaFor(service.name);
  const Icon = meta.icon;

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <div className="flex items-center gap-3">
            <span className="flex size-10 items-center justify-center rounded-lg bg-muted">
              <Icon className="size-5 text-muted-foreground" aria-hidden />
            </span>
            <div>
              <CardTitle className="text-base">{meta.label}</CardTitle>
              <p className="text-sm text-muted-foreground">{meta.blurb}</p>
            </div>
          </div>
        </CardHeader>
      </Card>
      <DefinitionSummary definition={service.definition} />
    </div>
  );
}
