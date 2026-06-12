import { useEffect, useState } from "react";
import {
  Boxes,
  Check,
  ChevronDown,
  ChevronRight,
  Loader2,
  Settings,
  Trash2,
} from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { useAsync } from "@/hooks/useAsync";
import { getProfileConfig } from "@/services/api";
import { cn } from "@/lib/utils";
import type { Profile } from "@/types/status";

import { onProfilesChanged } from "../profiles-events";
import type { ProfileActions } from "../hooks/useProfiles";

interface ProfileNavItemProps {
  profile: Profile;
  /** Current hash route, for highlighting the active service link. */
  route: string;
  actions: Pick<ProfileActions, "activate" | "remove">;
}

/**
 * One profile entry in the sidebar. Clicking the profile expands its submenu of
 * the project's services; per-profile actions set it active or delete it. The
 * active profile cannot be switched away from or removed (the backend refuses),
 * so both actions are hidden for it. Owns only its own expand/busy state.
 */
export function ProfileNavItem({
  profile,
  route,
  actions,
}: ProfileNavItemProps) {
  const [expanded, setExpanded] = useState(false);
  const [busy, setBusy] = useState(false);
  const [nonce, setNonce] = useState(0);

  // Refresh this profile's services when one changes elsewhere (e.g. added on
  // the Services screen), so the submenu stays in sync.
  useEffect(() => onProfilesChanged(() => setNonce((n) => n + 1)), []);

  // Load this profile's own services lazily, only once its submenu is open.
  const servicesState = useAsync(
    async (signal) =>
      expanded
        ? (await getProfileConfig(profile.name, signal)).services.map(
            (s) => s.name,
          )
        : [],
    [expanded, profile.name, nonce],
  );
  const services =
    servicesState.status === "success" ? servicesState.data : [];

  const settingsHref = `#/profiles/${encodeURIComponent(profile.name)}/settings`;
  const settingsActive = route === `/profiles/${profile.name}/settings`;

  const activate = async () => {
    setBusy(true);
    try {
      await actions.activate(profile.name);
      toast.success(`Switched to "${profile.name}"`);
    } catch (cause) {
      toast.error("Could not switch profile", {
        description: cause instanceof Error ? cause.message : String(cause),
      });
    } finally {
      setBusy(false);
    }
  };

  const remove = async () => {
    setBusy(true);
    try {
      await actions.remove(profile.name);
      toast.success(`Profile "${profile.name}" removed`);
    } catch (cause) {
      toast.error("Could not remove profile", {
        description: cause instanceof Error ? cause.message : String(cause),
      });
      setBusy(false);
    }
  };

  return (
    <li>
      <div className="group flex items-center gap-1 rounded-md pr-1 text-sm text-muted-foreground hover:bg-accent hover:text-accent-foreground">
        <button
          type="button"
          onClick={() => setExpanded((open) => !open)}
          aria-expanded={expanded}
          className="flex min-w-0 flex-1 items-center gap-1.5 rounded-md py-1.5 pl-2 text-left"
        >
          {expanded ? (
            <ChevronDown className="size-3.5 shrink-0" aria-hidden />
          ) : (
            <ChevronRight className="size-3.5 shrink-0" aria-hidden />
          )}
          <span className="truncate font-medium">{profile.name}</span>
          {profile.active && (
            <span
              className="size-1.5 shrink-0 rounded-full bg-emerald-500"
              aria-label="active"
            />
          )}
        </button>
        {!profile.active && (
          <div className="flex shrink-0 items-center opacity-0 transition-opacity focus-within:opacity-100 group-hover:opacity-100">
            <Button
              variant="ghost"
              size="icon"
              className="size-7"
              disabled={busy}
              aria-label={`Set profile ${profile.name} active`}
              title="Set active"
              onClick={activate}
            >
              {busy ? (
                <Loader2 className="animate-spin" aria-hidden />
              ) : (
                <Check aria-hidden />
              )}
            </Button>
            <ConfirmDialog
              trigger={
                <Button
                  variant="ghost"
                  size="icon"
                  className="size-7"
                  disabled={busy}
                  aria-label={`Delete profile ${profile.name}`}
                  title="Delete"
                >
                  <Trash2 aria-hidden />
                </Button>
              }
              title={`Delete "${profile.name}"?`}
              description="This removes the profile and its service config. This action cannot be undone."
              confirmLabel="Delete"
              variant="destructive"
              onConfirm={remove}
            />
          </div>
        )}
      </div>

      {expanded && (
        <ul className="mt-0.5 space-y-0.5 border-l border-border pl-3 ml-3">
          {servicesState.status === "loading" ? (
            <li className="px-2 py-1.5 text-xs text-muted-foreground">
              Loading…
            </li>
          ) : services.length === 0 ? (
            <li className="px-2 py-1.5 text-xs text-muted-foreground">
              No services defined.
            </li>
          ) : (
            services.map((name) => {
              const href = `#/profiles/${encodeURIComponent(profile.name)}/services/${encodeURIComponent(name)}`;
              const active = route === `/profiles/${profile.name}/services/${name}`;
              return (
                <li key={name}>
                  <a
                    href={href}
                    aria-current={active ? "page" : undefined}
                    className={cn(
                      "flex items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors",
                      active
                        ? "bg-accent text-accent-foreground"
                        : "text-muted-foreground hover:bg-accent hover:text-accent-foreground",
                    )}
                  >
                    <Boxes className="size-3.5 shrink-0" aria-hidden />
                    <span className="truncate font-mono text-xs">{name}</span>
                  </a>
                </li>
              );
            })
          )}
          <li>
            <a
              href={settingsHref}
              aria-current={settingsActive ? "page" : undefined}
              className={cn(
                "flex items-center gap-2 rounded-md px-2 py-1.5 text-sm transition-colors",
                settingsActive
                  ? "bg-accent text-accent-foreground"
                  : "text-muted-foreground hover:bg-accent hover:text-accent-foreground",
              )}
            >
              <Settings className="size-3.5 shrink-0" aria-hidden />
              <span className="truncate text-xs">Settings</span>
            </a>
          </li>
        </ul>
      )}
    </li>
  );
}
