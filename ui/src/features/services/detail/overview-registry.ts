import type { ComponentType } from "react";

import { MinioOverview } from "./MinioOverview";
import { OverviewPanel } from "./OverviewPanel";
import { PostgresOverview } from "./PostgresOverview";
import { RedisOverview } from "./RedisOverview";
import type { OverviewProps } from "./types";

/**
 * Per-service overview views, keyed by canonical service name. Postgres ships a
 * tailored overview; any service missing here falls back to the generic panel,
 * so a new backend service still renders without a code change — consistent
 * with the "don't special-case service names across the codebase" convention.
 * Add a richer view for another service by registering its component here.
 */
const OVERVIEWS: Record<string, ComponentType<OverviewProps>> = {
  postgres: PostgresOverview,
  minio: MinioOverview,
  redis: RedisOverview,
};

/** The overview component for a service name, or the generic fallback. */
export function overviewFor(name: string): ComponentType<OverviewProps> {
  return OVERVIEWS[name] ?? OverviewPanel;
}
