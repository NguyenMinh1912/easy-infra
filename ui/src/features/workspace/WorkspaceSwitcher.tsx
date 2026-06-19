import { useState } from "react";
import {
  Check,
  ChevronsUpDown,
  FolderOpen,
  Plus,
  TriangleAlert,
} from "lucide-react";
import { toast } from "sonner";

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

import { CreateWorkspaceDialog } from "./CreateWorkspaceDialog";
import { useWorkspaces } from "./hooks/useWorkspaces";

/** A full reload is the simplest correct way to refetch every screen's data
 * after the active folder changes — switching is a deliberate, infrequent act. */
function reloadApp() {
  window.location.reload();
}

/**
 * Sidebar control showing the active workspace with a dropdown to switch
 * between known workspaces and to create a new one. Lives at the top of the
 * sidebar so the active folder is always visible.
 */
export function WorkspaceSwitcher() {
  const { state, actions } = useWorkspaces();
  const [createOpen, setCreateOpen] = useState(false);

  const data = state.status === "success" ? state.data : null;
  const active = data?.workspaces.find((w) => w.name === data.active) ?? null;
  const label =
    state.status === "loading"
      ? "Loading…"
      : (active?.name ?? "No workspace");

  async function handleActivate(name: string) {
    if (name === data?.active) return;
    try {
      await actions.activate(name);
      reloadApp();
    } catch (cause) {
      toast.error("Could not switch workspace", {
        description: cause instanceof Error ? cause.message : String(cause),
      });
    }
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
            <FolderOpen className="size-4 shrink-0 text-muted-foreground" aria-hidden />
            <span className="truncate font-medium" title={active?.path}>
              {label}
            </span>
            <ChevronsUpDown
              className="ml-auto size-4 shrink-0 text-muted-foreground"
              aria-hidden
            />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent
          align="start"
          className="w-[14rem]"
        >
          <DropdownMenuLabel>Workspaces</DropdownMenuLabel>
          {data?.workspaces.map((w) => (
            <DropdownMenuItem
              key={w.name}
              onSelect={() => handleActivate(w.name)}
            >
              <Check
                className={cn(
                  "size-4",
                  w.name === data.active ? "opacity-100" : "opacity-0",
                )}
                aria-hidden
              />
              <span className="truncate" title={w.path}>
                {w.name}
              </span>
              {!w.exists && (
                <TriangleAlert
                  className="ml-auto size-4 text-destructive"
                  aria-label="folder missing"
                />
              )}
            </DropdownMenuItem>
          ))}
          <DropdownMenuSeparator />
          <DropdownMenuItem onSelect={() => setCreateOpen(true)}>
            <Plus className="size-4" aria-hidden />
            Create workspace…
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      <CreateWorkspaceDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        startPath={data?.home ?? ""}
        separator={data?.separator ?? "/"}
        onCreate={actions.create}
        onCreated={reloadApp}
      />
    </>
  );
}
