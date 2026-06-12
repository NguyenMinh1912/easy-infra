import { useEffect, useState } from "react";

/** Discriminated state of a one-shot async resource. */
export type AsyncState<T> =
  | { status: "loading" }
  | { status: "error"; error: Error }
  | { status: "success"; data: T };

/**
 * Run an abortable async loader on mount (and whenever `deps` change),
 * exposing a discriminated {@link AsyncState}. The loader receives an
 * `AbortSignal` and must propagate it so in-flight requests are cancelled on
 * unmount — preventing "set state after unmount" races.
 *
 * Generic on purpose: any endpoint can reuse it; it knows nothing about the
 * shape of `T` (open/closed, dependency-inversion).
 */
export function useAsync<T>(
  loader: (signal: AbortSignal) => Promise<T>,
  deps: React.DependencyList = [],
): AsyncState<T> {
  const [state, setState] = useState<AsyncState<T>>({ status: "loading" });

  useEffect(() => {
    const controller = new AbortController();
    setState({ status: "loading" });

    loader(controller.signal)
      .then((data) => {
        if (!controller.signal.aborted) {
          setState({ status: "success", data });
        }
      })
      .catch((cause: unknown) => {
        if (controller.signal.aborted) return;
        const error =
          cause instanceof Error ? cause : new Error(String(cause));
        setState({ status: "error", error });
      });

    return () => controller.abort();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);

  return state;
}
