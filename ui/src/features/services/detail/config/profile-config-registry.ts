import type { ComponentType } from "react";

import { MinioProfileConfigCard } from "./MinioProfileConfigCard";
import { PostgresProfileConfigCard } from "./PostgresProfileConfigCard";
import {
  ProfileServiceConfigCard,
  type ProfileServiceConfigCardProps,
} from "./ProfileServiceConfigCard";

/**
 * Per-service profile config editors, keyed by canonical service name. Postgres
 * ships a tailored editor (connection string ↔ fields, schema, connection
 * check) and minio adds a health-check button to the generic editor; any service
 * missing here falls back to the generic key/value card, so a new backend service
 * still renders without a UI change — mirroring the overview registry and the
 * "don't special-case service names" convention. Add a richer editor for another
 * service by registering its component here.
 */
const CARDS: Record<string, ComponentType<ProfileServiceConfigCardProps>> = {
  postgres: PostgresProfileConfigCard,
  minio: MinioProfileConfigCard,
};

/** The config card component for a service name, or the generic fallback. */
export function profileConfigCardFor(
  name: string,
): ComponentType<ProfileServiceConfigCardProps> {
  return CARDS[name] ?? ProfileServiceConfigCard;
}
