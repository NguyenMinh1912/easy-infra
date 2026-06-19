import { useCallback, useState } from "react";

import { DEFAULT_REGION } from "./regions";

/**
 * Persisted AWS region selection for the LocalStack overview. The dropdown is
 * the single source of truth: the choice is stored in localStorage (scoped per
 * profile/service so each can differ) and survives reload. When nothing is
 * stored yet it falls back to the profile's configured region, then the AWS
 * default.
 */
export function useRegion(
  scope: string,
  fallback?: string,
): [string, (region: string) => void] {
  const key = `easy-infra:localstack:region:${scope}`;

  const [region, setRegion] = useState<string>(() => {
    try {
      return (
        window.localStorage.getItem(key) ?? fallback ?? DEFAULT_REGION
      );
    } catch {
      return fallback ?? DEFAULT_REGION;
    }
  });

  const update = useCallback(
    (next: string) => {
      setRegion(next);
      try {
        window.localStorage.setItem(key, next);
      } catch {
        // Storage may be unavailable (private mode); the in-memory value still
        // drives the page, it just won't survive a reload.
      }
    },
    [key],
  );

  return [region, update];
}
