import { useState } from "react";
import { Loader2, RotateCcw } from "lucide-react";
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
import { notifyProfilesChanged } from "@/features/profiles";
import { updateService } from "@/services/api";
import type { ServiceConfig, ServiceInstance } from "@/types/service";

import { metaFor } from "../catalog-meta";
import { profileConfigCardFor } from "./config/profile-config-registry";
import type { ConfigRow } from "./config/ProfileServiceConfigCard";

interface ServiceSettingsDialogProps {
  /** Service whose settings are being edited. */
  service: ServiceInstance;
  /** Profile the service belongs to; the save and connection checks scope to it. */
  profile: string;
  /** Close the modal (the parent unmounts it). */
  onClose: () => void;
  /** Notify the parent that the service config changed, so it can reload. */
  onSaved?: () => void;
}

/**
 * Modal for editing one service's settings from the service detail navbar. Owns
 * the draft and reuses the same per-service config editor the profile screen
 * used (postgres connection string + check, minio health check, generic
 * key/value fallback), saving just this service via `updateService`. Reset
 * restores the values the modal opened with.
 */
export function ServiceSettingsDialog({
  service,
  profile,
  onClose,
  onSaved,
}: ServiceSettingsDialogProps) {
  const [rows, setRows] = useState<ConfigRow[]>(() => rowsFromConfig(service.config));
  const [saving, setSaving] = useState(false);

  const ConfigCard = profileConfigCardFor(service.type);
  const meta = metaFor(service.type);
  const Icon = meta.icon;

  const save = async () => {
    setSaving(true);
    try {
      await updateService(profile, service.id, configFromRows(rows));
      toast.success(`Settings for "${service.name}" saved`);
      notifyProfilesChanged();
      onSaved?.();
      onClose();
    } catch (cause) {
      toast.error("Could not save settings", {
        description: cause instanceof Error ? cause.message : String(cause),
      });
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <span className="flex size-8 items-center justify-center rounded-lg bg-muted">
              <Icon className="size-4 text-muted-foreground" aria-hidden />
            </span>
            {meta.label} settings
          </DialogTitle>
          <DialogDescription>
            Edit this service's settings for the{" "}
            <span className="font-mono">{profile}</span> profile.
          </DialogDescription>
        </DialogHeader>

        <div className="max-h-[60vh] overflow-y-auto pr-1">
          <ConfigCard
            name={service.type}
            profileName={profile}
            rows={rows}
            disabled={saving}
            onChange={setRows}
          />
        </div>

        <DialogFooter>
          <Button
            type="button"
            variant="ghost"
            disabled={saving}
            onClick={() => setRows(rowsFromConfig(service.config))}
          >
            <RotateCcw aria-hidden />
            Reset
          </Button>
          <Button type="button" variant="outline" disabled={saving} onClick={onClose}>
            Cancel
          </Button>
          <Button type="button" disabled={saving} onClick={save}>
            {saving && <Loader2 className="animate-spin" aria-hidden />}
            Save
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

/** Seed the draft rows from a service's saved config. */
function rowsFromConfig(config: ServiceConfig): ConfigRow[] {
  return Object.entries(config).map(([key, value]) => ({
    key,
    value: String(value),
  }));
}

/** Build the config object posted to the API, dropping rows with a blank key. */
function configFromRows(rows: ConfigRow[]): ServiceConfig {
  const config: ServiceConfig = {};
  for (const { key, value } of rows) {
    const trimmed = key.trim();
    if (trimmed) config[trimmed] = value;
  }
  return config;
}
