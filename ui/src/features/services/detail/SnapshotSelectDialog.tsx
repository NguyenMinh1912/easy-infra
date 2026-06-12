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
import { listSnapshots } from "@/services/api";
import { cn } from "@/lib/utils";

interface SnapshotSelectDialogProps {
  /** Service whose backup versions are offered. */
  serviceName: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Called with the chosen snapshot once the user confirms. */
  onApply: (snapshot: string) => void;
}

/**
 * Modal that lists a service's backup versions (newest first) and lets the user
 * pick which one to restore. Confirming hands the chosen snapshot to `onApply`,
 * which then streams the apply's progress; the picker itself does no applying.
 */
export function SnapshotSelectDialog({
  serviceName,
  open,
  onOpenChange,
  onApply,
}: SnapshotSelectDialogProps) {
  // Fetch fresh each time the dialog opens; resolve to nothing while closed so
  // the request only fires when the picker is actually shown.
  const state = useAsync(
    async (signal) =>
      open ? (await listSnapshots(serviceName, signal)).snapshots : [],
    [serviceName, open],
  );

  const [selected, setSelected] = useState("");

  // Default to the newest version once the list loads (or whenever it changes).
  useEffect(() => {
    if (state.status === "success" && state.data.length > 0) {
      setSelected((current) =>
        state.data.includes(current) ? current : state.data[0],
      );
    }
  }, [state]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Apply {serviceName}</DialogTitle>
          <DialogDescription>
            Choose which backup version to restore into the active profile's{" "}
            <span className="font-mono">{serviceName}</span>. You can follow the
            progress in the next step.
          </DialogDescription>
        </DialogHeader>

        <SnapshotList
          state={state}
          selected={selected}
          onSelect={setSelected}
        />

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            disabled={!selected}
            onClick={() => {
              onOpenChange(false);
              onApply(selected);
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
                Back this service up first — there is nothing to restore from.
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
