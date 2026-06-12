import { HardDrive } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

import { DefinitionSummary } from "./DefinitionSummary";
import { MinioBrowser } from "./browser/MinioBrowser";
import type { OverviewProps } from "./types";

/**
 * MinIO-specific overview. On a profile-scoped page it shows the object
 * browser — a read-only walk of that profile's buckets and the folders and
 * objects within them. Without a profile (no connection env) it falls back to
 * a summary of the service's config.
 */
export function MinioOverview({ service, profile }: OverviewProps) {
  if (!profile) {
    return <MinioSummary service={service} />;
  }
  return <MinioBrowser profile={profile} service={service.name} />;
}

/** The minio summary for the profile (version badge plus raw config). */
function MinioSummary({ service }: Pick<OverviewProps, "service">) {
  const version = String(service.config.version ?? "—");

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <div className="flex items-center gap-3">
            <span className="flex size-10 items-center justify-center rounded-lg bg-muted">
              <HardDrive className="size-5 text-muted-foreground" aria-hidden />
            </span>
            <div className="flex-1">
              <CardTitle className="text-base">MinIO</CardTitle>
              <p className="text-sm text-muted-foreground">
                S3-compatible object store
              </p>
            </div>
            <Badge variant="secondary">v{version}</Badge>
          </div>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">
            This profile owns this service. Its config — image version, the
            buckets to create, and the connection details — is edited under{" "}
            <span className="font-medium text-foreground">
              Profiles → Settings
            </span>
            .
          </p>
        </CardContent>
      </Card>
      <DefinitionSummary config={service.config} />
    </div>
  );
}
