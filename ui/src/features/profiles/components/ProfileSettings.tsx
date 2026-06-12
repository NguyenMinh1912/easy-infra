import { useState } from "react";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import type { ProfileConfig, ProfileServiceConfig } from "@/types/profiles";

import type { ProfileConfigActions } from "../hooks/useProfileConfig";
import {
  ProfileServiceConfigCard,
  type ConfigRow,
} from "./ProfileServiceConfigCard";

interface ProfileSettingsProps {
  data: ProfileConfig;
  actions: ProfileConfigActions;
}

/** Draft of one service's editable rows. */
interface ServiceDraft {
  name: string;
  rows: ConfigRow[];
}

/**
 * Editable view of a profile's per-service environment config. Owns the draft
 * for every service and saves them together in one request; the actual write
 * is delegated to the parent via `actions.save`. Reset restores the last saved
 * values.
 */
export function ProfileSettings({ data, actions }: ProfileSettingsProps) {
  const [draft, setDraft] = useState<ServiceDraft[]>(() => toDraft(data));
  const [saving, setSaving] = useState(false);

  const setRows = (name: string, rows: ConfigRow[]) =>
    setDraft((current) =>
      current.map((s) => (s.name === name ? { ...s, rows } : s)),
    );

  const save = async () => {
    setSaving(true);
    try {
      await actions.save(toServices(draft));
      toast.success(`Profile "${data.name}" saved`);
    } catch (cause) {
      toast.error("Could not save profile", {
        description: cause instanceof Error ? cause.message : String(cause),
      });
    } finally {
      setSaving(false);
    }
  };

  if (draft.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        This profile configures no services. Add a service on the Services
        screen first.
      </p>
    );
  }

  return (
    <div className="space-y-6">
      <div className="grid gap-4 sm:grid-cols-2">
        {draft.map((service) => (
          <ProfileServiceConfigCard
            key={service.name}
            name={service.name}
            rows={service.rows}
            disabled={saving}
            onChange={(rows) => setRows(service.name, rows)}
          />
        ))}
      </div>
      <div className="flex gap-2">
        <Button disabled={saving} onClick={save}>
          {saving && <Loader2 className="animate-spin" aria-hidden />}
          Save changes
        </Button>
        <Button
          variant="outline"
          disabled={saving}
          onClick={() => setDraft(toDraft(data))}
        >
          Reset
        </Button>
      </div>
    </div>
  );
}

function toDraft(data: ProfileConfig): ServiceDraft[] {
  return data.services.map((service) => ({
    name: service.name,
    rows: Object.entries(service.config).map(([key, value]) => ({
      key,
      value: String(value),
    })),
  }));
}

function toServices(draft: ServiceDraft[]): ProfileServiceConfig[] {
  return draft.map((service) => {
    const config: Record<string, unknown> = {};
    for (const { key, value } of service.rows) {
      const trimmed = key.trim();
      if (trimmed) {
        config[trimmed] = value;
      }
    }
    return { name: service.name, config };
  });
}
