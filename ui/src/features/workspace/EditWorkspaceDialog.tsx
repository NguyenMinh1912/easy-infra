import { useEffect, useState } from "react";
import { Loader2, Pencil } from "lucide-react";
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
import type { Workspace } from "@/types/workspace";

interface EditWorkspaceDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** The workspace being renamed (null when the dialog is closed). */
  workspace: Workspace | null;
  onRename: (id: number, name: string) => Promise<void>;
}

/** Rename-workspace dialog. The only editable field is the name. */
export function EditWorkspaceDialog({
  open,
  onOpenChange,
  workspace,
  onRename,
}: EditWorkspaceDialogProps) {
  const [name, setName] = useState("");
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (open && workspace) setName(workspace.name);
  }, [open, workspace]);

  const trimmed = name.trim();
  const canSubmit =
    trimmed.length > 0 && trimmed !== workspace?.name && !submitting;

  async function handleRename() {
    if (!canSubmit || !workspace) return;
    setSubmitting(true);
    try {
      await onRename(workspace.id, trimmed);
      toast.success(`Workspace renamed to "${trimmed}"`);
      onOpenChange(false);
    } catch (cause) {
      toast.error("Could not rename workspace", {
        description: cause instanceof Error ? cause.message : String(cause),
      });
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Rename workspace</DialogTitle>
          <DialogDescription>
            Give this workspace a new name. Its profiles and services are
            unchanged.
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-1.5">
          <label htmlFor="rename-workspace" className="text-sm font-medium">
            Name
          </label>
          <Input
            id="rename-workspace"
            value={name}
            onChange={(e) => setName(e.target.value)}
            disabled={submitting}
            autoFocus
            onKeyDown={(e) => {
              if (e.key === "Enter") handleRename();
            }}
          />
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
          <Button type="button" disabled={!canSubmit} onClick={handleRename}>
            {submitting ? (
              <Loader2 className="animate-spin" aria-hidden />
            ) : (
              <Pencil aria-hidden />
            )}
            {submitting ? "Saving…" : "Save"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
