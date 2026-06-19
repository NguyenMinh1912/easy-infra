import { useCallback, useEffect, useState } from "react";

import { getLocalstackHealth } from "@/services/api";
import type { HealthResponse } from "@/types/localstack";

/** How often to re-poll LocalStack health for live status, in milliseconds. */
const POLL_INTERVAL = 4000;

/**
 * Three connection states the overview renders: `loading` shows skeleton cards
 * on first load, `connected` carries the live health snapshot, and
 * `unreachable` explains LocalStack isn't responding and offers a retry.
 */
export type HealthState =
  | { status: "loading" }
  | { status: "connected"; health: HealthResponse }
  | { status: "unreachable"; error: string };

/**
 * Poll LocalStack health for the profile/service in the selected region. The
 * snapshot drives the service cards and Configuration panel. The first fetch
 * (and any region/profile change) shows `loading`; subsequent polls update in
 * place without flicker. Keeps polling even while unreachable so the page
 * recovers on its own when LocalStack comes back; `retry` forces an immediate
 * refetch.
 */
export function useLocalstackHealth(
  profile: string | undefined,
  service: string,
  region: string,
) {
  const [state, setState] = useState<HealthState>({ status: "loading" });
  const [nonce, setNonce] = useState(0);

  useEffect(() => {
    if (!profile) {
      setState({
        status: "unreachable",
        error: "Open this service from a profile to view its status.",
      });
      return;
    }

    let cancelled = false;
    let timer: ReturnType<typeof setTimeout> | null = null;
    const controller = new AbortController();
    setState({ status: "loading" });

    const poll = async () => {
      try {
        const health = await getLocalstackHealth(
          profile,
          service,
          region,
          controller.signal,
        );
        if (cancelled) return;
        setState(
          health.error
            ? { status: "unreachable", error: health.error }
            : { status: "connected", health },
        );
      } catch (cause) {
        if (cancelled || controller.signal.aborted) return;
        setState({
          status: "unreachable",
          error: cause instanceof Error ? cause.message : String(cause),
        });
      }
      if (!cancelled) timer = setTimeout(() => void poll(), POLL_INTERVAL);
    };
    void poll();

    return () => {
      cancelled = true;
      controller.abort();
      if (timer) clearTimeout(timer);
    };
  }, [profile, service, region, nonce]);

  const retry = useCallback(() => setNonce((n) => n + 1), []);
  return { state, retry };
}
