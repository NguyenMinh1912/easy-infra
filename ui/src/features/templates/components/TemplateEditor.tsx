import { useEffect, useState } from "react";
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
import { cn } from "@/lib/utils";
import { ApiError, createTemplate, getTemplate, updateTemplate } from "@/services/api";

import { parseVariables } from "../variables";

interface TemplateEditorProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Name of the template to edit; omit (or undefined) to create a new one. */
  editName?: string;
  /** Called after a successful save so the list can refresh. */
  onSaved: () => void;
}

/**
 * Create or edit a SQL template. In edit mode it loads the template's body on
 * open; the name is fixed once created (renames would orphan the row key), so
 * only description and SQL are editable.
 */
export function TemplateEditor({
  open,
  onOpenChange,
  editName,
  onSaved,
}: TemplateEditorProps) {
  const editing = editName !== undefined;
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [sql, setSql] = useState("");
  const [busy, setBusy] = useState(false);

  // Load the editable fields when the dialog opens: a blank form for create, or
  // the existing template's fields (body included) for edit.
  useEffect(() => {
    if (!open) return;
    if (!editing) {
      setName("");
      setDescription("");
      setSql("");
      return;
    }
    const controller = new AbortController();
    getTemplate(editName, controller.signal)
      .then((t) => {
        setName(t.name);
        setDescription(t.description);
        setSql(t.sql);
      })
      .catch((cause: unknown) => {
        if (controller.signal.aborted) return;
        toast.error("Could not load template", {
          description: cause instanceof ApiError ? cause.message : String(cause),
        });
      });
    return () => controller.abort();
  }, [open, editing, editName]);

  const variables = parseVariables(sql);

  const save = async () => {
    setBusy(true);
    try {
      if (editing) {
        await updateTemplate(editName, { description, sql });
      } else {
        await createTemplate({ name, description, sql });
      }
      toast.success(editing ? "Template updated" : "Template created");
      onOpenChange(false);
      onSaved();
    } catch (cause) {
      toast.error(editing ? "Update failed" : "Create failed", {
        description: cause instanceof ApiError ? cause.message : String(cause),
      });
    } finally {
      setBusy(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={(o) => !busy && onOpenChange(o)}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>{editing ? "Edit SQL template" : "New SQL template"}</DialogTitle>
          <DialogDescription>
            Save a reusable SQL script. Use{" "}
            <code className="rounded bg-muted px-1 py-0.5 font-mono">{"{{name}}"}</code>{" "}
            for values you fill in when you run it.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-1.5">
            <label htmlFor="tmpl-name" className="text-sm font-medium">
              Name
            </label>
            <Input
              id="tmpl-name"
              value={name}
              disabled={editing}
              placeholder="active-users"
              onChange={(e) => setName(e.target.value)}
            />
          </div>
          <div className="space-y-1.5">
            <label htmlFor="tmpl-desc" className="text-sm font-medium">
              Description
            </label>
            <Input
              id="tmpl-desc"
              value={description}
              placeholder="Users active since a given date"
              onChange={(e) => setDescription(e.target.value)}
            />
          </div>
          <div className="space-y-1.5">
            <label htmlFor="tmpl-sql" className="text-sm font-medium">
              SQL
            </label>
            <textarea
              id="tmpl-sql"
              value={sql}
              spellCheck={false}
              placeholder={"SELECT * FROM users\nWHERE last_seen >= {{since}};"}
              onChange={(e) => setSql(e.target.value)}
              className={cn(
                "flex min-h-40 w-full rounded-md border border-input bg-background px-3 py-2 font-mono text-sm shadow-sm transition-colors",
                "placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
              )}
            />
          </div>
          <p className="text-xs text-muted-foreground">
            Detected variables:{" "}
            {variables.length > 0 ? (
              <span className="font-mono text-foreground">{variables.join(", ")}</span>
            ) : (
              "none"
            )}
          </p>
        </div>

        <DialogFooter>
          <Button variant="outline" disabled={busy} onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            disabled={busy || name.trim() === "" || sql.trim() === ""}
            onClick={() => void save()}
          >
            {editing ? "Save changes" : "Create template"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
