import { useEffect, useState } from "react";
import { AlertCircle, Check, History } from "lucide-react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { useAsync } from "@/hooks/useAsync";
import { listProfiles, listSnapshots } from "@/services/api";
import { cn } from "@/lib/utils";

interface SnapshotSelectDialogProps {
  /** Service whose backup versions are offered. */
  serviceName: string;
  /** Profile the service is viewed under; scopes the snapshot lookup to it. */
  profile?: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /**
   * Called with the chosen snapshot and the profile it is restored from once the
   * user confirms. `sourceProfile` equals the viewed profile for a same-profile
   * restore, or another profile when restoring its backup of the same service.
   */
  onApply: (snapshot: string, sourceProfile: string) => void;
}

/**
 * Modal that lists a service's backup versions (newest first) and lets the user
 * pick which one to restore. A "restore from profile" selector lets the user
 * pull the same service's backup from another profile; the snapshot list tracks
 * the chosen profile. Confirming hands the chosen snapshot and source profile to
 * `onApply`, which then streams the apply's progress; the picker itself does no
 * applying.
 */
export function SnapshotSelectDialog({
  serviceName,
  profile,
  open,
  onOpenChange,
  onApply,
}: SnapshotSelectDialogProps) {
  // The profiles a backup may be restored from. Fetched when the dialog opens so
  // the selector reflects the current set; resolves to nothing while closed.
  const profiles = useAsync(
    async (signal) => (open ? await listProfiles(signal) : null),
    [open],
  );

  // Which profile's backups are listed. Defaults to the viewed profile (or the
  // active one when unset), so the dialog opens on a same-profile restore.
  const [source, setSource] = useState("");
  useEffect(() => {
    if (!open) return;
    if (profile) {
      setSource(profile);
    } else if (profiles.status === "success" && profiles.data) {
      setSource((current) => current || profiles.data!.activeProfile);
    }
  }, [open, profile, profiles]);

  // Fetch snapshots for the chosen source profile; resolve to nothing until both
  // the dialog is open and a source profile is known.
  const state = useAsync(
    async (signal) =>
      open && source
        ? (await listSnapshots(serviceName, source, signal)).snapshots
        : [],
    [serviceName, source, open],
  );

  const [selected, setSelected] = useState("");

  // Default to the newest version once the list loads (or whenever it changes).
  useEffect(() => {
    if (state.status === "success" && state.data.length > 0) {
      setSelected((current) =>
        state.data.includes(current) ? current : state.data[0],
      );
    } else {
      setSelected("");
    }
  }, [state]);

  const profileNames =
    profiles.status === "success" && profiles.data
      ? profiles.data.profiles.map((p) => p.name)
      : [];

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Apply {serviceName}</DialogTitle>
          <DialogDescription>
            Choose which backup version to restore into the viewed profile's{" "}
            <span className="font-mono">{serviceName}</span>. By default it
            restores this profile's own backup; pick another profile to restore
            its backup of the same service. You can follow the progress in the
            next step.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-1.5">
          <label htmlFor="apply-source-profile" className="text-sm font-medium">
            Restore from profile
          </label>
          <select
            id="apply-source-profile"
            value={source}
            onChange={(e) => setSource(e.target.value)}
            disabled={profileNames.length === 0}
            className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
          >
            {/* While profiles load, show the current source so the field is not empty. */}
            {profileNames.length === 0 && source && (
              <option value={source}>{source}</option>
            )}
            {profileNames.map((name) => (
              <option key={name} value={name}>
                {name}
                {name === profile ? " (current)" : ""}
              </option>
            ))}
          </select>
        </div>

        <SnapshotList state={state} selected={selected} onSelect={setSelected} />

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            disabled={!selected}
            onClick={() => {
              onOpenChange(false);
              onApply(selected, source);
            }}
          >
            Apply
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

interface SnapshotListProps {
  state: ReturnType<typeof useAsync<string[]>>;
  selected: string;
  onSelect: (snapshot: string) => void;
}

/** Renders the async snapshot list: skeleton, error, empty, or selectable rows. */
function SnapshotList({ state, selected, onSelect }: SnapshotListProps) {
  switch (state.status) {
    case "loading":
      return (
        <div className="space-y-2">
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
        </div>
      );
    case "error":
      return (
        <Alert variant="destructive">
          <AlertCircle />
          <div>
            <AlertTitle>Could not load backup versions</AlertTitle>
            <AlertDescription>{state.error.message}</AlertDescription>
          </div>
        </Alert>
      );
    case "success":
      if (state.data.length === 0) {
        return (
          <Alert>
            <History />
            <div>
              <AlertTitle>No backups yet</AlertTitle>
              <AlertDescription>
                This profile has no backups of the service — pick another
                profile or back it up first.
              </AlertDescription>
            </div>
          </Alert>
        );
      }
      return (
        <ul
          className="max-h-72 space-y-1 overflow-auto"
          role="radiogroup"
          aria-label="Backup version"
        >
          {state.data.map((snapshot, i) => {
            const active = snapshot === selected;
            return (
              <li key={snapshot}>
                <button
                  type="button"
                  role="radio"
                  aria-checked={active}
                  onClick={() => onSelect(snapshot)}
                  className={cn(
                    "flex w-full items-center justify-between rounded-md border px-3 py-2 text-left font-mono text-sm",
                    active
                      ? "border-primary bg-primary/5"
                      : "border-border hover:bg-muted/50",
                  )}
                >
                  <span className="flex items-center gap-2">
                    {snapshot}
                    {i === 0 && (
                      <span className="font-sans text-xs text-muted-foreground">
                        latest
                      </span>
                    )}
                  </span>
                  {active && <Check className="size-4 text-primary" aria-hidden />}
                </button>
              </li>
            );
          })}
        </ul>
      );
  }
}
