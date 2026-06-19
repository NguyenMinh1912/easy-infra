import { AlertCircle, ArrowLeft, ChevronRight, RotateCw } from "lucide-react";
import { useState } from "react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { useHashRoute } from "@/hooks/useHashRoute";
import type { ServiceConfig } from "@/types/service";

import {
  awsServiceFor,
  awsServiceMeta,
  hasDetail,
  type AwsServiceApp,
} from "./localstack/aws-services";
import { LocalstackConfigCard } from "./localstack/LocalstackConfigCard";
import { RegionSelect } from "./localstack/RegionSelect";
import { DEFAULT_REGION, formatRegion } from "./localstack/regions";
import { statusFor } from "./localstack/status";
import { useLocalstackHealth, type HealthState } from "./localstack/useLocalstackHealth";
import { useRegion } from "./localstack/useRegion";
import type { OverviewProps } from "./types";

/**
 * LocalStack overview. LocalStack emulates a bundle of AWS services, so this is
 * a live launcher: a region toolbar, then one status-driven card per service
 * read from `/_localstack/health` (polled), a reconciled Configuration panel,
 * and — when a service with a resource browser is opened via its own route — an
 * in-page detail view with a continuity header. Cards and config are both
 * driven by the same health snapshot, so they can't disagree.
 */
export function LocalstackOverview({ service, profile }: OverviewProps) {
  const route = useHashRoute();

  // Deep-linkable sub-route per AWS service, e.g.
  // #/profiles/{p}/services/{localstack}/sqs — selection lives in the URL so a
  // detail view is bookmarkable and the back button works.
  const basePath = profile
    ? `/profiles/${encodeURIComponent(profile)}/services/${encodeURIComponent(service.id)}`
    : null;
  const selectedId =
    basePath && route.startsWith(`${basePath}/`)
      ? decodeURIComponent(route.slice(basePath.length + 1).split("/")[0])
      : null;
  const selected = selectedId ? awsServiceFor(selectedId) : undefined;

  const navigate = (id: string | null) => {
    if (!basePath) return;
    window.location.hash = id ? `${basePath}/${encodeURIComponent(id)}` : basePath;
  };

  const configRegion = stringField(service.config.region) ?? DEFAULT_REGION;
  const [region, setRegion] = useRegion(
    `${profile ?? "_"}:${service.id}`,
    configRegion,
  );
  const [announcement, setAnnouncement] = useState("");
  const onRegionChange = (next: string) => {
    setRegion(next);
    setAnnouncement(`Region changed to ${formatRegion(next)}`);
  };

  const { state, retry } = useLocalstackHealth(profile, service.id, region);

  const version =
    (state.status === "connected" && state.health.version) ||
    stringField(service.config.version) ||
    "—";

  return (
    <div className="mx-auto w-full max-w-5xl space-y-6">
      <header className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <h1 className="text-lg font-semibold">LocalStack</h1>
          <Badge variant="secondary">v{version}</Badge>
        </div>
        <RegionSelect value={region} onChange={onRegionChange} />
        <span className="sr-only" aria-live="polite">
          {announcement}
        </span>
      </header>

      {selected?.Detail ? (
        <DetailView
          aws={selected}
          state={state}
          service={service}
          profile={profile}
          region={region}
          onBack={() => navigate(null)}
        />
      ) : (
        <LauncherView
          state={state}
          region={region}
          config={service.config}
          onOpen={navigate}
          onRetry={retry}
        />
      )}
    </div>
  );
}

/** The default view: service cards plus the Configuration panel. */
function LauncherView({
  state,
  region,
  config,
  onOpen,
  onRetry,
}: {
  state: HealthState;
  region: string;
  config: ServiceConfig;
  onOpen: (id: string) => void;
  onRetry: () => void;
}) {
  if (state.status === "loading") {
    return (
      <div className="grid gap-3 sm:grid-cols-2">
        {Array.from({ length: 6 }).map((_, i) => (
          <Skeleton key={i} className="h-[72px] w-full" />
        ))}
      </div>
    );
  }

  if (state.status === "unreachable") {
    return (
      <Alert variant="destructive">
        <AlertCircle />
        <div className="flex-1">
          <AlertTitle>LocalStack isn't responding</AlertTitle>
          <AlertDescription className="font-mono text-xs">
            {state.error}
          </AlertDescription>
          <Button
            variant="outline"
            size="sm"
            className="mt-3"
            onClick={onRetry}
          >
            <RotateCw aria-hidden />
            Retry
          </Button>
        </div>
      </Alert>
    );
  }

  const { health } = state;
  const ids = Object.keys(health.services).sort(
    (a, b) =>
      statusFor(health.services[a]).order - statusFor(health.services[b]).order ||
      a.localeCompare(b),
  );
  // Configuration "services" mirror what's actually emulated (not disabled),
  // so the panel can't disagree with the cards.
  const emulated = ids.filter(
    (id) => statusFor(health.services[id]).state !== "disabled",
  );

  return (
    <div className="space-y-6">
      <LocalstackConfigCard
        host={stringField(config.host) ?? "localhost"}
        port={stringField(config.port) ?? ""}
        region={region}
        version={health.version || stringField(config.version) || "—"}
        services={emulated}
      />

      {ids.length === 0 ? (
        <Card>
          <CardContent className="py-8 text-center text-sm text-muted-foreground">
            LocalStack reported no services.
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-3 sm:grid-cols-2">
          {ids.map((id) => (
            <ServiceCard
              key={id}
              id={id}
              state={health.services[id]}
              onOpen={onOpen}
            />
          ))}
        </div>
      )}

      <p className="text-sm text-muted-foreground">
        Which services LocalStack emulates is set per profile under{" "}
        <a
          href="#/profiles"
          className="font-medium text-foreground underline-offset-4 hover:underline focus-visible:underline focus-visible:outline-none"
        >
          Profiles → Settings
        </a>
        .
      </p>
    </div>
  );
}

/** One service card: icon, name, status, and (when browsable) a chevron. */
function ServiceCard({
  id,
  state,
  onOpen,
}: {
  id: string;
  state: string;
  onOpen: (id: string) => void;
}) {
  const meta = awsServiceMeta(id);
  const Icon = meta.icon;
  const status = statusFor(state);
  const browsable = hasDetail(id);

  const body = (
    <>
      <span className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-muted">
        <Icon className="size-5 text-muted-foreground" aria-hidden />
      </span>
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span className="font-medium text-foreground">{meta.label}</span>
          <StatusBadge state={state} />
        </div>
        <p className="truncate text-sm text-muted-foreground">{meta.blurb}</p>
      </div>
      {browsable && (
        <ChevronRight className="size-4 shrink-0 text-muted-foreground" aria-hidden />
      )}
    </>
  );

  const className =
    "flex items-center gap-3 rounded-lg border bg-card p-4 text-left transition-colors";

  if (!browsable) {
    return (
      <div className={className} aria-label={`${meta.name} — ${status.label}`}>
        {body}
      </div>
    );
  }

  return (
    <button
      type="button"
      onClick={() => onOpen(id)}
      aria-label={`Open ${meta.name} — ${status.label}`}
      className={`${className} hover:bg-accent focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none`}
    >
      {body}
    </button>
  );
}

/** Detail view for a browsable AWS service, with a continuity header. */
function DetailView({
  aws,
  state,
  service,
  profile,
  region,
  onBack,
}: {
  aws: AwsServiceApp;
  state: HealthState;
  service: OverviewProps["service"];
  profile?: string;
  region: string;
  onBack: () => void;
}) {
  const Detail = aws.Detail!;
  const Icon = aws.icon;
  const liveState =
    state.status === "connected" ? state.health.services[aws.id] : undefined;

  return (
    <div className="space-y-6">
      <Button
        variant="ghost"
        size="sm"
        className="-ml-2 gap-1.5 text-muted-foreground"
        onClick={onBack}
      >
        <ArrowLeft className="size-4" aria-hidden />
        LocalStack
      </Button>
      <Card>
        <CardContent className="flex items-center gap-3 py-4">
          <span className="flex size-10 items-center justify-center rounded-lg bg-muted">
            <Icon className="size-5 text-muted-foreground" aria-hidden />
          </span>
          <div className="flex-1">
            <div className="flex items-center gap-2">
              <span className="font-medium text-foreground">{aws.name}</span>
              {liveState && <StatusBadge state={liveState} />}
            </div>
            <p className="text-sm text-muted-foreground">{aws.blurb}</p>
          </div>
        </CardContent>
      </Card>
      <Detail service={service} profile={profile} region={region} />
    </div>
  );
}

/** A status dot + label badge reflecting a service's health state. */
function StatusBadge({ state }: { state: string }) {
  const status = statusFor(state);
  return (
    <span
      className="inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-xs font-medium text-foreground"
      title={status.label}
    >
      <span
        className={`size-2 rounded-full ${status.dotClass} ${
          status.state === "running" ? "motion-safe:animate-pulse" : ""
        }`}
        aria-hidden
      />
      {status.label}
    </span>
  );
}

/**
 * Read a config value as a non-empty trimmed string, or undefined. The config
 * is a free-form string/number map, so values may arrive as either.
 */
function stringField(value: unknown): string | undefined {
  if (value === undefined || value === null) return undefined;
  const s = String(value).trim();
  return s === "" ? undefined : s;
}
