import { useEffect, useState } from "react";
import {
  Check,
  ChevronDown,
  ChevronRight,
  Loader2,
  Pencil,
  Plus,
  Settings,
  Trash2,
} from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { metaFor, ServiceDialog, type DialogState } from "@/features/services";
import { useAsync } from "@/hooks/useAsync";
import {
  createService,
  deleteService,
  getProfileConfig,
  getServiceCatalog,
  updateService,
} from "@/services/api";
import { cn } from "@/lib/utils";
import type { CatalogEntry, ServiceConfig, ServiceInstance } from "@/types/service";
import type { Profile } from "@/types/status";

import { notifyProfilesChanged, onProfilesChanged } from "../profiles-events";
import type { ProfileActions } from "../hooks/useProfiles";

interface ProfileNavItemProps {
  profile: Profile;
  /** Current hash route, for highlighting the active service link. */
  route: string;
  actions: Pick<ProfileActions, "activate" | "remove">;
}

/** The profile's services together with the catalog of available ones. */
interface SubmenuData {
  services: ServiceInstance[];
  catalog: CatalogEntry[];
}

/**
 * One profile entry in the sidebar. Clicking the profile expands its submenu of
 * the project's services, where they can be added, edited, and removed inline —
 * services belong to a profile, so this is where they are managed. Per-profile
 * actions set it active or delete it. The active profile cannot be switched away
 * from or removed (the backend refuses), so both actions are hidden for it. Owns
 * only its own expand/busy state.
 */
export function ProfileNavItem({
  profile,
  route,
  actions,
}: ProfileNavItemProps) {
  const [expanded, setExpanded] = useState(false);
  const [busy, setBusy] = useState(false);
  const [mutating, setMutating] = useState(false);
  const [dialog, setDialog] = useState<DialogState | null>(null);
  const [nonce, setNonce] = useState(0);

  // Refresh this profile's services when one changes elsewhere (e.g. via the
  // service detail screen), so the submenu stays in sync.
  useEffect(() => onProfilesChanged(() => setNonce((n) => n + 1)), []);

  // Load this profile's services and the service catalog lazily, only once its
  // submenu is open. The catalog drives the "add service" choices.
  const dataState = useAsync<SubmenuData | null>(
    async (signal) => {
      if (!expanded) return null;
      const [config, catalog] = await Promise.all([
        getProfileConfig(profile.name, signal),
        getServiceCatalog(signal),
      ]);
      return { services: config.services, catalog: catalog.services };
    },
    [expanded, profile.name, nonce],
  );
  const data = dataState.status === "success" ? dataState.data : null;
  const services = data?.services ?? [];
  const catalog = data?.catalog ?? [];
  const defined = new Set(services.map((s) => s.name));
  const available = catalog.filter((entry) => !defined.has(entry.name));

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

  // Run a service mutation: toast the outcome and, on success, notify so this
  // submenu (and any other view of this profile) reloads.
  const run = async (
    action: () => Promise<unknown>,
    messages: { success: string; error: string },
  ): Promise<boolean> => {
    setMutating(true);
    try {
      await action();
      toast.success(messages.success);
      notifyProfilesChanged();
      return true;
    } catch (cause) {
      toast.error(messages.error, {
        description: cause instanceof Error ? cause.message : String(cause),
      });
      return false;
    } finally {
      setMutating(false);
    }
  };

  // Create with defaults, then persist any edits the user made in the dialog.
  const add = (name: string, config: ServiceConfig) => {
    const defaults =
      catalog.find((entry) => entry.name === name)?.defaultConfig ?? {};
    const edited = !sameConfig(config, defaults);
    return run(
      async () => {
        await createService(profile.name, name);
        if (edited) await updateService(profile.name, name, config);
      },
      { success: `Service "${name}" added`, error: `Could not add "${name}"` },
    );
  };

  const submit = async (name: string, config: ServiceConfig) => {
    if (!dialog) return;
    const ok =
      dialog.mode === "add"
        ? await add(name, config)
        : await run(() => updateService(profile.name, name, config), {
            success: `Service "${name}" updated`,
            error: `Could not update "${name}"`,
          });
    if (ok) setDialog(null);
  };

  const removeService = (name: string) =>
    run(() => deleteService(profile.name, name), {
      success: `Service "${name}" removed`,
      error: `Could not remove "${name}"`,
    });

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
          {dataState.status === "loading" ? (
            <li className="px-2 py-1.5 text-xs text-muted-foreground">
              Loading…
            </li>
          ) : dataState.status === "error" ? (
            <li className="px-2 py-1.5 text-xs text-destructive">
              Could not load services.
            </li>
          ) : (
            <>
              {services.length === 0 ? (
                <li className="px-2 py-1.5 text-xs text-muted-foreground">
                  No services defined.
                </li>
              ) : (
                services.map((service) => {
                  const href = `#/profiles/${encodeURIComponent(profile.name)}/services/${encodeURIComponent(service.name)}`;
                  const active =
                    route ===
                    `/profiles/${profile.name}/services/${service.name}`;
                  const Icon = metaFor(service.name).icon;
                  return (
                    <li key={service.name}>
                      <div
                        className={cn(
                          "group/svc flex items-center gap-1 rounded-md pr-1 transition-colors",
                          active
                            ? "bg-accent text-accent-foreground"
                            : "text-muted-foreground hover:bg-accent hover:text-accent-foreground",
                        )}
                      >
                        <a
                          href={href}
                          aria-current={active ? "page" : undefined}
                          className="flex min-w-0 flex-1 items-center gap-2 rounded-md px-2 py-1.5 text-sm"
                        >
                          <Icon className="size-3.5 shrink-0" aria-hidden />
                          <span className="truncate font-mono text-xs">
                            {service.name}
                          </span>
                        </a>
                        <div className="flex shrink-0 items-center opacity-0 transition-opacity focus-within:opacity-100 group-hover/svc:opacity-100">
                          <Button
                            variant="ghost"
                            size="icon"
                            className="size-6"
                            disabled={mutating}
                            aria-label={`Edit service ${service.name}`}
                            title="Edit"
                            onClick={() => setDialog({ mode: "edit", service })}
                          >
                            <Pencil aria-hidden />
                          </Button>
                          <ConfirmDialog
                            trigger={
                              <Button
                                variant="ghost"
                                size="icon"
                                className="size-6"
                                disabled={mutating}
                                aria-label={`Remove service ${service.name}`}
                                title="Remove"
                              >
                                <Trash2 aria-hidden />
                              </Button>
                            }
                            title={`Remove "${service.name}"?`}
                            description={`This removes ${service.name} from the "${profile.name}" profile. This action cannot be undone.`}
                            confirmLabel="Remove"
                            variant="destructive"
                            onConfirm={() => removeService(service.name)}
                          />
                        </div>
                      </div>
                    </li>
                  );
                })
              )}
              <li>
                <button
                  type="button"
                  disabled={mutating || available.length === 0}
                  title={
                    available.length === 0
                      ? "All supported services are already defined"
                      : undefined
                  }
                  onClick={() => setDialog({ mode: "add" })}
                  className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50"
                >
                  <Plus className="size-3.5 shrink-0" aria-hidden />
                  <span className="text-xs">Add service</span>
                </button>
              </li>
            </>
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

      {dialog && (
        <ServiceDialog
          dialog={dialog}
          available={available}
          catalog={catalog}
          busy={mutating}
          onClose={() => setDialog(null)}
          onSubmit={submit}
        />
      )}
    </li>
  );
}

/** Whether two definitions are equal once values are compared as strings. */
function sameConfig(a: ServiceConfig, b: ServiceConfig): boolean {
  const keys = Object.keys(a);
  if (keys.length !== Object.keys(b).length) return false;
  return keys.every((key) => key in b && String(a[key]) === String(b[key]));
}
