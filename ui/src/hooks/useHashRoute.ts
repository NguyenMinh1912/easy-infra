import { useSyncExternalStore } from "react";

/**
 * Subscribe to the current client-side route, derived from `location.hash`.
 * The app uses hash routing so the embedded SPA needs no server-side route
 * handling — every path resolves to index.html and the hash selects the screen.
 *
 * Returns the path portion after `#`, defaulting to `/` (e.g. `/`, `/services`).
 */
export function useHashRoute(): string {
  return useSyncExternalStore(subscribe, getRoute, getRoute);
}

function subscribe(onChange: () => void): () => void {
  window.addEventListener("hashchange", onChange);
  return () => window.removeEventListener("hashchange", onChange);
}

function getRoute(): string {
  return window.location.hash.replace(/^#/, "") || "/";
}
