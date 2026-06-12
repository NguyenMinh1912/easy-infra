import { useAsync } from "@/hooks/useAsync";
import { getStatus } from "@/services/api";
import type { Status } from "@/types/status";

/**
 * Load the project status for the dashboard. Thin feature binding over the
 * generic {@link useAsync} + the `getStatus` service — keeps components free of
 * data-fetching concerns.
 */
export function useStatus() {
  return useAsync<Status>((signal) => getStatus(signal));
}
