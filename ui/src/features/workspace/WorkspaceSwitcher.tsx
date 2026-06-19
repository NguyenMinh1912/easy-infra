import { useRef, useState } from "react";
import {
  Check,
  ChevronsUpDown,
  Download,
  FolderOpen,
  Pencil,
  Plus,
  Trash2,
  Upload,
} from "lucide-react";
import { toast } from "sonner";

import { exportWorkspaceUrl } from "@/services/api";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";
import type { Workspace } from "@/types/workspace";

import { CreateWorkspaceDialog } from "./CreateWorkspaceDialog";
import { EditWorkspaceDialog } from "./EditWorkspaceDialog";
import { useWorkspaces } from "./hooks/useWorkspaces";

/** A full reload is the simplest correct way to refetch every screen's data
 * after the active workspace changes — switching is a deliberate, infrequent
 * act. */
function reloadApp() {
  window.location.reload();
}

/**
 * Sidebar control showing the active workspace with a dropdown to switch
 * between known workspaces, rename or remove them, and create a new one.
 */
export function WorkspaceSwitcher() {
  const { state, actions } = useWorkspaces();
  const [createOpen, setCreateOpen] = useState(false);
  const [editing, setEditing] = useState<Workspace | null>(null);
  const importInputRef = useRef<HTMLInputElement>(null);

  const data = state.status === "success" ? state.data : null;
  const active = data?.workspaces.find((w) => w.id === data.active) ?? null;
  const label =
    state.status === "loading" ? "Loading…" : (active?.name ?? "No workspace");

  async function handleActivate(id: number) {
    if (id === data?.active) return;
    try {
      await actions.activate(id);
      reloadApp();
    } catch (cause) {
      toast.error("Could not switch workspace", {
        description: cause instanceof Error ? cause.message : String(cause),
      });
    }
  }

  async function handleRemove(w: Workspace) {
    try {
      await actions.remove(w.id);
      // Removing the active workspace changes the data behind every screen.
      if (w.id === data?.active) reloadApp();
    } catch (cause) {
      toast.error("Could not remove workspace", {
        description: cause instanceof Error ? cause.message : String(cause),
      });
    }
  }

  function handleExport(w: Workspace) {
    // The endpoint sends the file with a Content-Disposition header, so a
    // hidden anchor click lets the browser save it without leaving the app.
    const a = document.createElement("a");
    a.href = exportWorkspaceUrl(w.id);
    a.download = `${w.name}.json`;
    document.body.appendChild(a);
    a.click();
    a.remove();
  }

  async function handleImportFile(file: File) {
    try {
      await actions.importFile(file);
      // The imported workspace becomes active, so reload to reflect it app-wide.
      reloadApp();
    } catch (cause) {
      toast.error("Could not import workspace", {
        description: cause instanceof Error ? cause.message : String(cause),
      });
    }
  }

  async function handleRename(id: number, name: string) {
    await actions.rename(id, name);
    // A rename changes the active workspace's display everywhere; reload if it
    // was the active one so the whole app reflects the new name.
    if (id === data?.active) reloadApp();
  }

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="outline"
            className="h-auto w-full justify-start gap-2 px-3 py-2 text-left"
            disabled={state.status === "loading"}
          >
            <FolderOpen
              className="size-4 shrink-0 text-muted-foreground"
              aria-hidden
            />
            <span className="truncate font-medium">{label}</span>
            <ChevronsUpDown
              className="ml-auto size-4 shrink-0 text-muted-foreground"
              aria-hidden
            />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start" className="w-[16rem]">
          <DropdownMenuLabel>Workspaces</DropdownMenuLabel>
          {data?.workspaces.map((w) => (
            <DropdownMenuItem
              key={w.id}
              onSelect={() => handleActivate(w.id)}
              className="gap-2"
            >
              <Check
                className={cn(
                  "size-4 shrink-0",
                  w.id === data.active ? "opacity-100" : "opacity-0",
                )}
                aria-hidden
              />
              <span className="truncate">{w.name}</span>
              <span className="ml-auto flex items-center gap-0.5">
                <button
                  type="button"
                  className="rounded p-1 text-muted-foreground hover:bg-accent hover:text-foreground"
                  aria-label={`Export ${w.name}`}
                  onClick={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    handleExport(w);
                  }}
                >
                  <Download className="size-3.5" aria-hidden />
                </button>
                <button
                  type="button"
                  className="rounded p-1 text-muted-foreground hover:bg-accent hover:text-foreground"
                  aria-label={`Rename ${w.name}`}
                  onClick={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    setEditing(w);
                  }}
                >
                  <Pencil className="size-3.5" aria-hidden />
                </button>
                <button
                  type="button"
                  className="rounded p-1 text-muted-foreground hover:bg-accent hover:text-destructive"
                  aria-label={`Remove ${w.name}`}
                  onClick={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    handleRemove(w);
                  }}
                >
                  <Trash2 className="size-3.5" aria-hidden />
                </button>
              </span>
            </DropdownMenuItem>
          ))}
          <DropdownMenuSeparator />
          <DropdownMenuItem onSelect={() => setCreateOpen(true)}>
            <Plus className="size-4" aria-hidden />
            Create workspace…
          </DropdownMenuItem>
          <DropdownMenuItem onSelect={() => importInputRef.current?.click()}>
            <Upload className="size-4" aria-hidden />
            Import workspace…
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      <input
        ref={importInputRef}
        type="file"
        accept="application/json,.json"
        className="hidden"
        onChange={(e) => {
          const file = e.target.files?.[0];
          // Reset so selecting the same file again still fires onChange.
          e.target.value = "";
          if (file) void handleImportFile(file);
        }}
      />

      <CreateWorkspaceDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        onCreate={actions.create}
        onCreated={reloadApp}
      />

      <EditWorkspaceDialog
        open={editing !== null}
        onOpenChange={(open) => {
          if (!open) setEditing(null);
        }}
        workspace={editing}
        onRename={handleRename}
      />
    </>
  );
}
