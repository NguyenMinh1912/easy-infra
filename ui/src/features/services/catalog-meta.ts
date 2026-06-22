import { Box, Cloud, Database, HardDrive, Hammer, Zap } from "lucide-react";
import type { LucideIcon } from "lucide-react";

/**
 * Presentation metadata for a service. This layer is frontend-only and purely
 * cosmetic — the raw service `name` from the API stays the source of truth. It
 * exists so the UI can show a friendly label, a one-line blurb, and an icon per
 * service without the backend having to supply any of that.
 */
export interface ServiceMeta {
  /** Friendly display label, e.g. "Postgres". */
  label: string;
  /** One-line description shown under the name. */
  blurb: string;
  icon: LucideIcon;
}

/**
 * Known services easy-infra supports. Keyed by the canonical service name. Any
 * name missing here falls back to a generic entry (see {@link metaFor}), so a
 * new backend service still renders without a code change — consistent with the
 * "don't special-case service names across the codebase" convention.
 */
const META: Record<string, ServiceMeta> = {
  postgres: {
    label: "Postgres",
    blurb: "Relational database",
    icon: Database,
  },
  redis: {
    label: "Redis",
    blurb: "In-memory data store",
    icon: Zap,
  },
  minio: {
    label: "MinIO",
    blurb: "S3-compatible object store",
    icon: HardDrive,
  },
  localstack: {
    label: "LocalStack",
    blurb: "AWS cloud emulator",
    icon: Cloud,
  },
  jenkins: {
    label: "Jenkins",
    blurb: "CI/CD automation server",
    icon: Hammer,
  },
};

/**
 * Presentation metadata for a service name, falling back to a generic entry
 * (the raw name as label, a neutral icon) for services not in {@link META}.
 */
export function metaFor(name: string): ServiceMeta {
  return META[name] ?? { label: name, blurb: "Service", icon: Box };
}
