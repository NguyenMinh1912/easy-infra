import { useState } from "react";
import { Check, Trash2 } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import type { Profile } from "@/types/status";

interface ProfileRowProps {
  profile: Profile;
  onActivate: (name: string) => Promise<void>;
  onRemove: (name: string) => Promise<void>;
}

/**
 * One profile row: its name, an active flag, and per-row actions to set it
 * active or delete it. Owns only its own busy/error state; the active profile
 * cannot be switched away from or removed here (the backend refuses), so both
 * actions are disabled for it.
 */
export function ProfileRow({ profile, onActivate, onRemove }: ProfileRowProps) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function run(action: (name: string) => Promise<void>) {
    setBusy(true);
    setError(null);
    try {
      await action(profile.name);
      // On success the list reloads and this row unmounts; leave busy set.
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause));
      setBusy(false);
    }
  }

  return (
    <li className="py-3">
      <div className="flex items-center justify-between gap-3">
        <span className="flex items-center gap-2 text-sm">
          {profile.name}
          {profile.active && <Badge variant="success">active</Badge>}
        </span>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={profile.active || busy}
            onClick={() => run(onActivate)}
          >
            <Check aria-hidden />
            Set active
          </Button>
          <Button
            variant="ghost"
            size="sm"
            disabled={profile.active || busy}
            onClick={() => run(onRemove)}
            aria-label={`Delete profile ${profile.name}`}
          >
            <Trash2 aria-hidden />
            Delete
          </Button>
        </div>
      </div>
      {error && (
        <p role="alert" className="mt-2 text-sm text-destructive">
          {error}
        </p>
      )}
    </li>
  );
}
