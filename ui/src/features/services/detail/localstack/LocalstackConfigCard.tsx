import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

import { CopyButton } from "../redis/CopyButton";
import { formatRegion } from "./regions";

interface LocalstackConfigCardProps {
  /** Connection host from the profile env. */
  host: string;
  /** Connection port from the profile env. */
  port: string;
  /** Region selected in the toolbar (the single source of truth). */
  region: string;
  /** Version reported by the health response (falls back to config). */
  version: string;
  /** Service ids the emulator actually reports (drives the cards too). */
  services: string[];
}

/**
 * LocalStack Configuration panel, reconciled with the live health response so
 * it can't disagree with the cards or the version badge: one endpoint built
 * from host + port (copyable), the selected region (copyable), the reported
 * version, and the set of emulated services. Replaces the generic key/value
 * summary for LocalStack.
 */
export function LocalstackConfigCard({
  host,
  port,
  region,
  version,
  services,
}: LocalstackConfigCardProps) {
  const endpoint = port ? `${host}:${port}` : host;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Configuration</CardTitle>
      </CardHeader>
      <CardContent>
        <dl className="grid gap-x-8 gap-y-3 sm:grid-cols-2">
          <ConfigRow label="Endpoint" value={endpoint} copyLabel="endpoint" />
          <ConfigRow
            label="Region"
            value={region}
            display={formatRegion(region)}
            copyLabel="region"
          />
          <div className="flex items-center justify-between gap-4 border-b border-border/50 pb-1.5">
            <dt className="text-sm font-medium text-foreground">Version</dt>
            <dd className="font-mono text-sm text-foreground">{version}</dd>
          </div>
          <div className="flex items-start justify-between gap-4 border-b border-border/50 pb-1.5">
            <dt className="text-sm font-medium text-foreground">Services</dt>
            <dd className="font-mono text-sm text-foreground">
              {services.length > 0 ? services.join(", ") : "—"}
            </dd>
          </div>
        </dl>
      </CardContent>
    </Card>
  );
}

/** A label/value row with a copy affordance for the value. */
function ConfigRow({
  label,
  value,
  display,
  copyLabel,
}: {
  label: string;
  value: string;
  display?: string;
  copyLabel: string;
}) {
  return (
    <div className="flex items-center justify-between gap-2 border-b border-border/50 pb-1.5">
      <dt className="text-sm font-medium text-foreground">{label}</dt>
      <dd className="flex min-w-0 items-center gap-1">
        <span className="truncate font-mono text-sm text-foreground">
          {display ?? value}
        </span>
        <CopyButton value={value} label={copyLabel} className="size-6" />
      </dd>
    </div>
  );
}
