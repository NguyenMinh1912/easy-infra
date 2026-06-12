import { useState, type FormEvent } from "react";
import {
  ChevronDown,
  ChevronRight,
  Layers,
  Loader2,
  Plus,
  X,
} from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useAsync } from "@/hooks/useAsync";
import { useHashRoute } from "@/hooks/useHashRoute";
import { listServices } from "@/services/api";
import { cn } from "@/lib/utils";

import { useProfiles } from "../hooks/useProfiles";
import { ProfileNavItem } from "./ProfileNavItem";

/**
 * Sidebar "Profiles" section: an expandable group listing the project's
 * profiles, with inline create/activate/delete. Each profile expands to the
 * project's service list as submenu items. Owns its own data via
 * {@link useProfiles}; the service names are loaded once and shared across
 * profiles since service definitions are project-wide.
 */
export function ProfileNav() {
  const route = useHashRoute();
  const { state, actions } = useProfiles();
  const services = useAsync((signal) => listServices(signal), []);

  const [open, setOpen] = useState(true);
  const [creating, setCreating] = useState(false);
  const [name, setName] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const profiles = state.status === "success" ? state.data.profiles : [];
  const serviceNames =
    services.status === "success"
      ? services.data.services.map((s) => s.name)
      : [];
  const groupActive = route === "/profiles" || route.startsWith("/profiles/");

  const trimmed = name.trim();
  const duplicate = profiles.some(
    (p) => p.name.toLowerCase() === trimmed.toLowerCase(),
  );
  const canSubmit = trimmed.length > 0 && !duplicate && !submitting;

  async function handleCreate(event: FormEvent) {
    event.preventDefault();
    if (!canSubmit) return;
    setSubmitting(true);
    try {
      await actions.create(trimmed);
      toast.success(`Profile "${trimmed}" created`);
      setName("");
      setCreating(false);
      setOpen(true);
    } catch (cause) {
      toast.error("Could not create profile", {
        description: cause instanceof Error ? cause.message : String(cause),
      });
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div>
      <div
        className={cn(
          "flex items-center gap-1 rounded-md pr-1 text-sm font-medium transition-colors",
          groupActive
            ? "bg-accent text-accent-foreground"
            : "text-muted-foreground hover:bg-accent hover:text-accent-foreground",
        )}
      >
        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          aria-expanded={open}
          aria-label={open ? "Collapse profiles" : "Expand profiles"}
          className="rounded-md py-2 pl-3"
        >
          {open ? (
            <ChevronDown className="size-4 shrink-0" aria-hidden />
          ) : (
            <ChevronRight className="size-4 shrink-0" aria-hidden />
          )}
        </button>
        <a
          href="#/profiles"
          aria-current={groupActive ? "page" : undefined}
          className="flex min-w-0 flex-1 items-center gap-3 py-2"
        >
          <Layers className="size-4 shrink-0" aria-hidden />
          Profiles
        </a>
        <Button
          variant="ghost"
          size="icon"
          className="size-7"
          aria-label={creating ? "Cancel new profile" : "New profile"}
          title="New profile"
          onClick={() => {
            setCreating((v) => !v);
            setOpen(true);
          }}
        >
          {creating ? <X aria-hidden /> : <Plus aria-hidden />}
        </Button>
      </div>

      {open && (
        <div className="mt-1 space-y-0.5 pl-4">
          {creating && (
            <form onSubmit={handleCreate} className="flex items-center gap-1 py-1">
              <label htmlFor="sidebar-profile-name" className="sr-only">
                Profile name
              </label>
              <Input
                id="sidebar-profile-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="new-profile"
                disabled={submitting}
                aria-invalid={trimmed.length > 0 && duplicate}
                autoFocus
                className="h-8 text-sm"
              />
              <Button
                type="submit"
                size="icon"
                className="size-8 shrink-0"
                disabled={!canSubmit}
                aria-label="Create profile"
              >
                {submitting ? (
                  <Loader2 className="animate-spin" aria-hidden />
                ) : (
                  <Plus aria-hidden />
                )}
              </Button>
            </form>
          )}

          {state.status === "loading" && (
            <p className="px-2 py-1.5 text-xs text-muted-foreground">Loading…</p>
          )}
          {state.status === "error" && (
            <p className="px-2 py-1.5 text-xs text-destructive">
              Could not load profiles.
            </p>
          )}
          {state.status === "success" &&
            (profiles.length === 0 ? (
              <p className="px-2 py-1.5 text-xs text-muted-foreground">
                No profiles yet.
              </p>
            ) : (
              <ul className="space-y-0.5">
                {profiles.map((profile) => (
                  <ProfileNavItem
                    key={profile.name}
                    profile={profile}
                    services={serviceNames}
                    route={route}
                    actions={actions}
                  />
                ))}
              </ul>
            ))}
        </div>
      )}
    </div>
  );
}
