import { useState } from "react";
import { Check, Loader2, Trash2 } from "lucide-react";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import type { Profile } from "@/types/status";

interface ProfileRowProps {
  profile: Profile;
  onActivate: (name: string) => Promise<void>;
  onRemove: (name: string) => Promise<void>;
}

/**
 * One profile row: its name, an active flag, and per-row actions to set it
 * active or delete it. Owns only its own busy state; success/failure are
 * surfaced as toasts. The active profile cannot be switched away from or
 * removed here (the backend refuses), so both actions are disabled for it.
 */
export function ProfileRow({ profile, onActivate, onRemove }: ProfileRowProps) {
  const [busy, setBusy] = useState(false);

  const activate = async () => {
    setBusy(true);
    try {
      await onActivate(profile.name);
      toast.success(`Switched to "${profile.name}"`);
      // On success the list reloads and this row unmounts; leave busy set.
    } catch (cause) {
      toast.error("Could not switch profile", {
        description: cause instanceof Error ? cause.message : String(cause),
      });
      setBusy(false);
    }
  };

  const remove = async () => {
    setBusy(true);
    try {
      await onRemove(profile.name);
      toast.success(`Profile "${profile.name}" removed`);
    } catch (cause) {
      toast.error("Could not remove profile", {
        description: cause instanceof Error ? cause.message : String(cause),
      });
      setBusy(false);
    }
  };

  return (
    <li className="flex items-center justify-between gap-3 py-3">
      <span className="flex items-center gap-2 text-sm font-medium">
        {profile.name}
        {profile.active && <Badge variant="success">active</Badge>}
      </span>
      <div className="flex items-center gap-2">
        <Button
          variant="outline"
          size="sm"
          disabled={profile.active || busy}
          onClick={activate}
        >
          {busy ? (
            <Loader2 className="animate-spin" aria-hidden />
          ) : (
            <Check aria-hidden />
          )}
          Set active
        </Button>
        <ConfirmDialog
          trigger={
            <Button
              variant="ghost"
              size="sm"
              disabled={profile.active || busy}
              aria-label={`Delete profile ${profile.name}`}
            >
              <Trash2 aria-hidden />
              Delete
            </Button>
          }
          title={`Delete "${profile.name}"?`}
          description="This removes the profile and its service config. This action cannot be undone."
          confirmLabel="Delete"
          variant="destructive"
          onConfirm={remove}
        />
      </div>
    </li>
  );
}
