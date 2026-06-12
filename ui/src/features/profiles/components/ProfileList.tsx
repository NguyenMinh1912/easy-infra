import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import type { Profile } from "@/types/status";

import { ProfileRow } from "./ProfileRow";

interface ProfileListProps {
  profiles: Profile[];
  onActivate: (name: string) => Promise<void>;
  onRemove: (name: string) => Promise<void>;
}

/** Lists the project's profiles, each with set-active and delete actions. */
export function ProfileList({
  profiles,
  onActivate,
  onRemove,
}: ProfileListProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm font-medium uppercase tracking-wide text-muted-foreground">
          Profiles
        </CardTitle>
      </CardHeader>
      <CardContent>
        {profiles.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            No profiles yet. Add one above.
          </p>
        ) : (
          <ul className="divide-y divide-border">
            {profiles.map((profile) => (
              <ProfileRow
                key={profile.name}
                profile={profile}
                onActivate={onActivate}
                onRemove={onRemove}
              />
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}
