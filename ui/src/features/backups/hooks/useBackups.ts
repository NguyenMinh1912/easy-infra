import { useCallback, useState } from "react";

import { useAsync } from "@/hooks/useAsync";
import { listBackups, type BackupList } from "@/services/api";

/** How many backup sessions to show per page. */
export const PAGE_SIZE = 10;

/**
 * Load a page of backup sessions. Owns the current page and exposes `setPage`
 * to navigate plus `reload` so a delete (or a backup finishing) can refresh the
 * current page after it succeeds.
 */
export function useBackups(pageSize = PAGE_SIZE) {
  const [page, setPage] = useState(1);
  const [reloadKey, setReloadKey] = useState(0);
  const reload = useCallback(() => setReloadKey((k) => k + 1), []);

  const state = useAsync<BackupList>(
    (signal) => listBackups(page, pageSize, signal),
    [page, pageSize, reloadKey],
  );

  return { state, page, setPage, pageSize, reload };
}
