import { Database } from "lucide-react";
import { lazy, Suspense } from "react";

import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

import { DefinitionSummary } from "./DefinitionSummary";
import type { OverviewProps } from "./types";

// Loaded on demand: the console pulls in CodeMirror, which would otherwise
// dominate the main bundle for users who never open it.
const PostgresConsoleTabs = lazy(() =>
  import("./console/PostgresConsoleTabs").then((m) => ({
    default: m.PostgresConsoleTabs,
  })),
);

// Loaded on demand: the relationship canvas pulls in React Flow, kept out of
// the main bundle for users who never open it.
const RelationGraph = lazy(() =>
  import("./graph/RelationGraph").then((m) => ({ default: m.RelationGraph })),
);

/**
 * Postgres-specific overview. Surfaces the image version prominently and lists
 * the profile's raw config (version plus connection settings). On a
 * profile-scoped page it offers two views against the profile's connection: the
 * SQL console and a relationship canvas for exploring foreign keys visually.
 */
export function PostgresOverview({ service, profile }: OverviewProps) {
  if (!profile) {
    return <PostgresSummary service={service} />;
  }
  return (
    <Tabs defaultValue="console" className="space-y-4">
      <TabsList>
        <TabsTrigger value="console">SQL Console</TabsTrigger>
        <TabsTrigger value="graph">Relationships</TabsTrigger>
      </TabsList>
      <TabsContent value="console">
        <Suspense fallback={<Skeleton className="h-40 w-full" />}>
          <PostgresConsoleTabs profile={profile} service={service.id} />
        </Suspense>
      </TabsContent>
      <TabsContent value="graph">
        <Suspense fallback={<Skeleton className="h-[70vh] w-full" />}>
          <RelationGraph profile={profile} service={service.id} />
        </Suspense>
      </TabsContent>
    </Tabs>
  );
}

/** The postgres summary for the profile (version badge plus raw config). */
function PostgresSummary({ service }: Pick<OverviewProps, "service">) {
  const version = String(service.config.version ?? "—");

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
            This profile owns this service. Its config — image version plus the
            connection details (host, port, credentials, database) — is edited
            under{" "}
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
