import { useCallback, useEffect, useState } from "react";
import {
  ArrowUp,
  Folder,
  FolderGit2,
  Loader2,
  Plus,
} from "lucide-react";
import { toast } from "sonner";

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
import { browseDirs } from "@/services/api";
import type { DirListing } from "@/types/workspace";

interface CreateWorkspaceDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Initial folder to browse from (typically the user's home directory). */
  startPath: string;
  /** OS path separator, used to compose the new workspace's folder path. */
  separator: string;
  onCreate: (name: string, path: string) => Promise<void>;
  /** Called after a workspace is created so the app can reload. */
  onCreated: () => void;
}

/**
 * Create-workspace dialog. The user names the workspace and navigates the
 * server's filesystem to choose where it lives; the new project folder is
 * `<browsed folder>/<name>`. The backend scaffolds the project on create.
 */
export function CreateWorkspaceDialog({
  open,
  onOpenChange,
  startPath,
  separator,
  onCreate,
  onCreated,
}: CreateWorkspaceDialogProps) {
  const [name, setName] = useState("");
  const [listing, setListing] = useState<DirListing | null>(null);
  const [loading, setLoading] = useState(false);
  const [browseError, setBrowseError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const navigate = useCallback((path?: string) => {
    setLoading(true);
    setBrowseError(null);
    browseDirs(path)
      .then(setListing)
      .catch((cause: unknown) =>
        setBrowseError(cause instanceof Error ? cause.message : String(cause)),
      )
      .finally(() => setLoading(false));
  }, []);

  // (Re)start browsing from startPath each time the dialog opens.
  useEffect(() => {
    if (open) {
      setName("");
      navigate(startPath || undefined);
    }
  }, [open, startPath, navigate]);

  const trimmed = name.trim();
  const targetPath =
    listing && trimmed ? `${listing.path}${separator}${trimmed}` : "";
  const canSubmit = trimmed.length > 0 && listing !== null && !submitting;

  async function handleCreate() {
    if (!canSubmit || !listing) return;
    setSubmitting(true);
    try {
      await onCreate(trimmed, targetPath);
      toast.success(`Workspace "${trimmed}" created`);
      onOpenChange(false);
      onCreated();
    } catch (cause) {
      toast.error("Could not create workspace", {
        description: cause instanceof Error ? cause.message : String(cause),
      });
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Create workspace</DialogTitle>
          <DialogDescription>
            Name the workspace and choose the folder it should live in. A new
            project is scaffolded inside it.
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <label htmlFor="workspace-name" className="text-sm font-medium">
              Name
            </label>
            <Input
              id="workspace-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. my-app"
              disabled={submitting}
            />
          </div>

          <div className="flex flex-col gap-1.5">
            <span className="text-sm font-medium">Location</span>
            <div className="rounded-md border">
              <div className="flex items-center gap-2 border-b px-3 py-2">
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="size-7 shrink-0"
                  disabled={!listing?.parent || loading}
                  onClick={() => navigate(listing?.parent)}
                  aria-label="Parent folder"
                >
                  <ArrowUp className="size-4" aria-hidden />
                </Button>
                <span
                  className="truncate font-mono text-xs text-muted-foreground"
                  title={listing?.path}
                >
                  {listing?.path ?? "…"}
                </span>
              </div>
              <div className="h-48 overflow-y-auto p-1">
                {loading ? (
                  <div className="flex h-full items-center justify-center text-muted-foreground">
                    <Loader2 className="size-4 animate-spin" aria-hidden />
                  </div>
                ) : browseError ? (
                  <p className="p-3 text-sm text-destructive">{browseError}</p>
                ) : listing && listing.entries.length === 0 ? (
                  <p className="p-3 text-sm text-muted-foreground">
                    No subfolders here.
                  </p>
                ) : (
                  listing?.entries.map((entry) => (
                    <button
                      key={entry.path}
                      type="button"
                      onClick={() => navigate(entry.path)}
                      className="flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-left text-sm transition-colors hover:bg-accent hover:text-accent-foreground"
                    >
                      {entry.isProject ? (
                        <FolderGit2
                          className="size-4 shrink-0 text-muted-foreground"
                          aria-hidden
                        />
                      ) : (
                        <Folder
                          className="size-4 shrink-0 text-muted-foreground"
                          aria-hidden
                        />
                      )}
                      <span className="truncate">{entry.name}</span>
                      {entry.isProject && (
                        <span className="ml-auto shrink-0 text-xs text-muted-foreground">
                          project
                        </span>
                      )}
                    </button>
                  ))
                )}
              </div>
            </div>
            {targetPath && (
              <p className="font-mono text-xs text-muted-foreground" title={targetPath}>
                Creates <span className="text-foreground">{targetPath}</span>
              </p>
            )}
          </div>
        </div>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={submitting}
          >
            Cancel
          </Button>
          <Button type="button" disabled={!canSubmit} onClick={handleCreate}>
            {submitting ? (
              <Loader2 className="animate-spin" aria-hidden />
            ) : (
              <Plus aria-hidden />
            )}
            {submitting ? "Creating…" : "Create workspace"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
