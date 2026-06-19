import { AlertCircle } from "lucide-react";
import type { ReactNode } from "react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { useAsync } from "@/hooks/useAsync";
import { checkServiceConnection } from "@/services/api";
import type { ServiceInstance } from "@/types/service";

import { metaFor } from "../catalog-meta";

/**
 * Service types whose health can be probed over the wire. Mirrors the services
 * with a `Health` implementation server-side (postgres, minio, redis); other
 * types have no `Health` implementation server-side, so probing them would
 * report a false "not active". Kept as a single set rather than scattered name
 * checks, in line with the "don't special-case service names" convention.
 */
const HEALTH_CHECKABLE = new Set(["postgres", "minio", "redis"]);

interface ServiceHealthBannerProps {
  /** Service whose health is probed. */
  service: ServiceInstance;
  /** Profile scoping the probe; without it (or for an unprobeable type) no check runs. */
  profile?: string;
  /**
   * The service detail UI to show once the service is confirmed reachable.
   * When the service is not active, it is withheld so the user sees only the
   * warning — not a detail screen backed by a service that cannot answer.
   */
  children: ReactNode;
}

/**
 * On entering a service detail page, probe the service's health. While checking,
 * when healthy, or for service types without a health probe, the detail UI
 * ({@link children}) is shown as-is. When the service is not active (unreachable
 * / not responding), the detail UI is withheld and a destructive alert is shown
 * in its place — so the user is told the status without a detail screen that
 * would only error against a service that cannot respond.
 */
export function ServiceHealthBanner({ service, profile, children }: ServiceHealthBannerProps) {
  const probe = Boolean(profile) && HEALTH_CHECKABLE.has(service.type);

  const state = useAsync(
    async (signal) => {
      if (!probe) return { ok: true };
      return checkServiceConnection(profile!, service.type, service.config, signal);
    },
    [probe, profile, service.type, service.config],
  );

  if (!probe || state.status === "loading") {
    return <>{children}</>;
  }

  // A transport failure means we could not confirm the service is up; treat it,
  // like an explicit `ok: false`, as "not active" so the user is told either way.
  const reason =
    state.status === "error"
      ? state.error.message
      : state.data.ok
        ? null
        : (state.data.error ?? "The service did not respond to a health check.");

  if (!reason) {
    return <>{children}</>;
  }

  const label = metaFor(service.type).label;

  return (
    <Alert variant="destructive">
      <AlertCircle />
      <div>
        <AlertTitle>{label} is not active</AlertTitle>
        <AlertDescription>
          {reason}. The service may be stopped or unreachable — start it (or
          review its connection settings under <strong>Settings</strong>), then
          refresh.
        </AlertDescription>
      </div>
    </Alert>
  );
}
