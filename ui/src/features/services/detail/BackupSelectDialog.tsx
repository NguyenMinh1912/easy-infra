import { useEffect, useState } from "react";
import { AlertCircle, Check } from "lucide-react";

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
import { getBackupOptions, type BackupOptions } from "@/services/api";
import { cn } from "@/lib/utils";

interface BackupSelectDialogProps {
  /** Service to back up; its buckets (if any) are offered for selection. */
  serviceName: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /**
   * Called once the user confirms. `buckets` is the chosen subset for an
   * object store; it is omitted when the service has no buckets, meaning "back
   * up everything".
   */
  onBackup: (buckets?: string[]) => void;
}

/**
 * Modal shown when backing up a service. For an object store (minio) it lists
 * the buckets a backup can capture and lets the user pick which to include,
 * defaulting to the buckets declared in the profile settings. Services without
 * a bucket concept get a plain confirmation. Confirming hands the selection to
 * `onBackup`, which then streams the snapshot's progress.
 */
export function BackupSelectDialog({
  serviceName,
  open,
  onOpenChange,
  onBackup,
}: BackupSelectDialogProps) {
  // Fetch fresh each time the dialog opens; resolve to nothing while closed so
  // the request only fires when the dialog is actually shown.
  const state = useAsync<BackupOptions | null>(
    async (signal) => (open ? await getBackupOptions(serviceName, signal) : null),
    [serviceName, open],
  );

  const [selected, setSelected] = useState<string[]>([]);

  // Seed the selection from the server's default once the options load.
  useEffect(() => {
    if (state.status === "success" && state.data) {
      setSelected(state.data.selected);
    }
  }, [state]);

  const options = state.status === "success" ? state.data : null;
  const hasBuckets = !!options && options.buckets.length > 0;
  // With buckets on offer, require at least one so deselecting all cannot
  // silently fall back to backing up everything.
  const canBackUp = state.status === "success" && (!hasBuckets || selected.length > 0);

  const toggle = (bucket: string) =>
    setSelected((current) =>
      current.includes(bucket)
        ? current.filter((b) => b !== bucket)
        : [...current, bucket],
    );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Back up {serviceName}</DialogTitle>
          <DialogDescription>
            {hasBuckets ? (
              <>
                Choose which buckets to snapshot into the active profile's{" "}
                <span className="font-mono">{serviceName}</span> backup. Defaults
                to the buckets in your profile settings.
              </>
            ) : (
              <>
                This snapshots the active profile's{" "}
                <span className="font-mono">{serviceName}</span> data into a new
                backup folder. You can follow the progress in the next step.
              </>
            )}
          </DialogDescription>
        </DialogHeader>

        <BucketList
          state={state}
          selected={selected}
          onToggle={toggle}
        />

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            disabled={!canBackUp}
            onClick={() => {
              onOpenChange(false);
              onBackup(hasBuckets ? selected : undefined);
            }}
          >
            Back up
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

interface BucketListProps {
  state: ReturnType<typeof useAsync<BackupOptions | null>>;
  selected: string[];
  onToggle: (bucket: string) => void;
}

/** Renders the async bucket options: skeleton, error, or a selectable list. */
function BucketList({ state, selected, onToggle }: BucketListProps) {
  switch (state.status) {
    case "loading":
      return (
        <div className="space-y-2">
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
        </div>
      );
    case "error":
      return (
        <Alert variant="destructive">
          <AlertCircle />
          <div>
            <AlertTitle>Could not load buckets</AlertTitle>
            <AlertDescription>{state.error.message}</AlertDescription>
          </div>
        </Alert>
      );
    case "success": {
      const options = state.data;
      // No bucket concept (e.g. postgres): nothing to choose, the footer
      // confirms a back-up-everything.
      if (!options || options.buckets.length === 0) {
        return options?.error ? (
          <Alert variant="destructive">
            <AlertCircle />
            <div>
              <AlertTitle>Could not list buckets</AlertTitle>
              <AlertDescription>{options.error}</AlertDescription>
            </div>
          </Alert>
        ) : null;
      }
      return (
        <>
          {options.error && (
            <Alert variant="destructive">
              <AlertCircle />
              <div>
                <AlertTitle>Showing configured buckets only</AlertTitle>
                <AlertDescription>{options.error}</AlertDescription>
              </div>
            </Alert>
          )}
          <ul
            className="max-h-72 space-y-1 overflow-auto"
            role="group"
            aria-label="Buckets to back up"
          >
            {options.buckets.map((bucket) => {
              const active = selected.includes(bucket);
              return (
                <li key={bucket}>
                  <button
                    type="button"
                    role="checkbox"
                    aria-checked={active}
                    onClick={() => onToggle(bucket)}
                    className={cn(
                      "flex w-full items-center justify-between rounded-md border px-3 py-2 text-left font-mono text-sm",
                      active
                        ? "border-primary bg-primary/5"
                        : "border-border hover:bg-muted/50",
                    )}
                  >
                    <span>{bucket}</span>
                    {active && (
                      <Check className="size-4 text-primary" aria-hidden />
                    )}
                  </button>
                </li>
              );
            })}
          </ul>
        </>
      );
    }
  }
}
