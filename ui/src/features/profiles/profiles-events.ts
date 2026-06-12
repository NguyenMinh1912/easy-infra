// A tiny in-app pub/sub so a profile mutation made outside the profiles UI —
// notably a "fork to local", which creates the `local` profile from the service
// detail screen — can ask the sidebar's profile list to refresh. Each component
// owns its own data via useAsync, so without this signal the new profile would
// only appear after a manual reload.

type Listener = () => void;

const listeners = new Set<Listener>();

/** Subscribe to profile-list changes; returns an unsubscribe function. */
export function onProfilesChanged(listener: Listener): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

/** Notify subscribers that the project's profiles changed and should reload. */
export function notifyProfilesChanged(): void {
  listeners.forEach((listener) => listener());
}
