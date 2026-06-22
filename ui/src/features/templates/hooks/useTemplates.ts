import { useCallback, useState } from "react";

import { useAsync } from "@/hooks/useAsync";
import { listTemplates } from "@/services/api";
import type { TemplateSummary } from "@/types/templates";

/**
 * Load the workspace's SQL templates. Exposes `reload` so a create, edit, or
 * delete can refresh the list after it succeeds.
 */
export function useTemplates() {
  const [reloadKey, setReloadKey] = useState(0);
  const reload = useCallback(() => setReloadKey((k) => k + 1), []);

  const state = useAsync<TemplateSummary[]>(
    (signal) => listTemplates(signal),
    [reloadKey],
  );

  return { state, reload };
}
