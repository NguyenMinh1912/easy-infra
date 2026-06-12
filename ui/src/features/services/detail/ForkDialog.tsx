import { useEffect, useState } from "react";
import { AlertCircle, Check, Sparkles } from "lucide-react";

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
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { useAsync } from "@/hooks/useAsync";
import { getProfileConfig, listProfiles, listSnapshots } from "@/services/api";
import { cn } from "@/lib/utils";

interface ForkDialogProps {
  /** Service whose backup versions are offered as fork seeds. */
  serviceName: string;
  /** Source profile the service is viewed under; scopes the fork to it. */
  profile?: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /**
   * Called with the chosen snapshot, and the local port to publish the forked
   * container on, once the user confirms. An empty snapshot means "create a
   * fresh backup of the source first, then fork from it"; an undefined port
   * keeps the source profile's port.
   */
  onFork: (snapshot: string, port?: number) => void;
}

/** Sentinel for the "create a new backup" choice — the server reads "" the same way. */
const NEW_BACKUP = "";

/**
 * Modal for "fork to local": pick which backup version seeds the new local
 * container, or take a fresh backup of the source. Defaults to a fresh backup
 * (the first row) so a user can fork without having backed up first; confirming
 * hands the choice to `onFork`, which then streams the fork's progress.
 */
export function ForkDialog({
  serviceName,
  profile,
  open,
  onOpenChange,
  onFork,
}: ForkDialogProps) {
  // Fetch fresh each time the dialog opens; resolve to nothing while closed so
  // the request only fires when the picker is actually shown.
  const state = useAsync(
    async (signal) =>
      open ? (await listSnapshots(serviceName, profile, signal)).snapshots : [],
    [serviceName, profile, open],
  );

  // Default to a fresh backup; the user can switch to an existing version.
  const [selected, setSelected] = useState<string>(NEW_BACKUP);
  useEffect(() => {
    if (open) setSelected(NEW_BACKUP);
  }, [open]);

  // The local container defaults to the source profile's port; fetch it so the
  // field can be pre-filled with the value the user is editing.
  const sourcePort = useAsync(
    async (signal) => {
      if (!open) return "";
      // Scope to the viewed profile; fall back to the active one when unset.
      const sourceProfile =
        profile ?? (await listProfiles(signal)).activeProfile;
      const cfg = await getProfileConfig(sourceProfile, signal);
      const raw = cfg.services.find((s) => s.name === serviceName)?.config.port;
      return raw == null ? "" : String(raw);
    },
    [serviceName, profile, open],
  );

  const [port, setPort] = useState("");
  useEffect(() => {
    if (open && sourcePort.status === "success") setPort(sourcePort.data);
  }, [open, sourcePort]);

  // A blank field keeps the source port; otherwise the entered number is sent.
  const trimmedPort = port.trim();
  const portNumber = Number(trimmedPort);
  const portInvalid =
    trimmedPort !== "" &&
    (!Number.isInteger(portNumber) || portNumber < 1 || portNumber > 65535);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Fork {serviceName} to local</DialogTitle>
          <DialogDescription>
            Launch a local container with the active profile's{" "}
            <span className="font-mono">{serviceName}</span> configuration and
            seed it from a backup. Pick a version to fork from, or take a fresh
            backup of the source first.
          </DialogDescription>
        </DialogHeader>

        <ForkSourceList
          state={state}
          selected={selected}
          onSelect={setSelected}
        />

        <div className="space-y-1.5">
          <label htmlFor="fork-local-port" className="text-sm font-medium">
            Local port
          </label>
          <Input
            id="fork-local-port"
            type="number"
            inputMode="numeric"
            min={1}
            max={65535}
            value={port}
            onChange={(e) => setPort(e.target.value)}
            placeholder="Same as source"
            aria-invalid={portInvalid}
          />
          <p
            className={cn(
              "text-xs",
              portInvalid ? "text-destructive" : "text-muted-foreground",
            )}
          >
            {portInvalid
              ? "Enter a port between 1 and 65535, or leave blank to keep the source's."
              : "Port the local container is published on. Leave blank to keep the source's."}
          </p>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            disabled={portInvalid}
            onClick={() => {
              onOpenChange(false);
              onFork(selected, trimmedPort === "" ? undefined : portNumber);
            }}
          >
            Fork
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

interface ForkSourceListProps {
  state: ReturnType<typeof useAsync<string[]>>;
  selected: string;
  onSelect: (snapshot: string) => void;
}

/**
 * The selectable seed list: a "create a new backup" row is always offered first,
 * followed by the existing backup versions (newest first) once they load.
 */
function ForkSourceList({ state, selected, onSelect }: ForkSourceListProps) {
  return (
    <ul
      className="max-h-72 space-y-1 overflow-auto"
      role="radiogroup"
      aria-label="Fork source"
    >
      <li>
        <SourceRow
          active={selected === NEW_BACKUP}
          onSelect={() => onSelect(NEW_BACKUP)}
        >
          <span className="flex items-center gap-2 font-sans">
            <Sparkles className="size-4 text-primary" aria-hidden />
            Create a new backup
          </span>
        </SourceRow>
      </li>

      {state.status === "loading" && (
        <>
          <li>
            <Skeleton className="h-10 w-full" />
          </li>
          <li>
            <Skeleton className="h-10 w-full" />
          </li>
        </>
      )}
      {state.status === "error" && (
        <li>
          <Alert variant="destructive">
            <AlertCircle />
            <div>
              <AlertTitle>Could not load backup versions</AlertTitle>
              <AlertDescription>{state.error.message}</AlertDescription>
            </div>
          </Alert>
        </li>
      )}
      {state.status === "success" &&
        state.data.map((snapshot, i) => (
          <li key={snapshot}>
            <SourceRow
              active={snapshot === selected}
              onSelect={() => onSelect(snapshot)}
            >
              <span className="flex items-center gap-2">
                {snapshot}
                {i === 0 && (
                  <span className="font-sans text-xs text-muted-foreground">
                    latest
                  </span>
                )}
              </span>
            </SourceRow>
          </li>
        ))}
    </ul>
  );
}

interface SourceRowProps {
  active: boolean;
  onSelect: () => void;
  children: React.ReactNode;
}

/** One selectable row in the fork source list. */
function SourceRow({ active, onSelect, children }: SourceRowProps) {
  return (
    <button
      type="button"
      role="radio"
      aria-checked={active}
      onClick={onSelect}
      className={cn(
        "flex w-full items-center justify-between rounded-md border px-3 py-2 text-left font-mono text-sm",
        active
          ? "border-primary bg-primary/5"
          : "border-border hover:bg-muted/50",
      )}
    >
      {children}
      {active && <Check className="size-4 text-primary" aria-hidden />}
    </button>
  );
}
