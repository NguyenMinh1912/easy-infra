import { Zap } from "lucide-react";
import { lazy, Suspense } from "react";

import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

import { DefinitionSummary } from "./DefinitionSummary";
import type { OverviewProps } from "./types";

// Loaded on demand: the key browser and console are only needed once a user
// opens a profile-scoped Redis page.
const RedisKeyBrowser = lazy(() =>
  import("./redis/RedisKeyBrowser").then((m) => ({ default: m.RedisKeyBrowser })),
);
const RedisConsole = lazy(() =>
  import("./redis/RedisConsole").then((m) => ({ default: m.RedisConsole })),
);

/**
 * Redis-specific overview. On a profile-scoped page it offers two tabs against
 * that profile's connection: a key browser (SCAN by pattern, value viewer per
 * type) and a redis-cli-style command console. Without a profile (no connection
 * env) it falls back to a summary of the service's config.
 */
export function RedisOverview({ service, profile }: OverviewProps) {
  if (!profile) {
    return <RedisSummary service={service} />;
  }
  return (
    <Tabs defaultValue="keys">
      <TabsList variant="line">
        <TabsTrigger value="keys">Keys</TabsTrigger>
        <TabsTrigger value="console">Console</TabsTrigger>
      </TabsList>
      <TabsContent value="keys" className="mt-4">
        <Suspense fallback={<Skeleton className="h-48 w-full" />}>
          <RedisKeyBrowser profile={profile} service={service.id} />
        </Suspense>
      </TabsContent>
      <TabsContent value="console" className="mt-4">
        <Suspense fallback={<Skeleton className="h-40 w-full" />}>
          <RedisConsole profile={profile} service={service.id} />
        </Suspense>
      </TabsContent>
    </Tabs>
  );
}

/** The redis summary for the profile (version badge plus raw config). */
function RedisSummary({ service }: Pick<OverviewProps, "service">) {
  const version = String(service.config.version ?? "—");

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <div className="flex items-center gap-3">
            <span className="flex size-10 items-center justify-center rounded-lg bg-muted">
              <Zap className="size-5 text-muted-foreground" aria-hidden />
            </span>
            <div className="flex-1">
              <CardTitle className="text-base">Redis</CardTitle>
              <p className="text-sm text-muted-foreground">
                In-memory data store
              </p>
            </div>
            <Badge variant="secondary">v{version}</Badge>
          </div>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">
            This profile owns this service. Its config — image version plus the
            connection details (host, port, password, database) — is edited
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
