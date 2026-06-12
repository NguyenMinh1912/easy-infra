import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import type { Profile } from "@/types/status";

interface ProfilesCardProps {
  profiles: Profile[];
}

/** Lists the project's profiles, flagging the active one. */
export function ProfilesCard({ profiles }: ProfilesCardProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm font-medium uppercase tracking-wide text-muted-foreground">
          Profiles
        </CardTitle>
      </CardHeader>
      <CardContent>
        {profiles.length === 0 ? (
          <p className="text-sm text-muted-foreground">No profiles yet.</p>
        ) : (
          <ul className="divide-y divide-border">
            {profiles.map((profile) => (
              <li
                key={profile.name}
                className="flex items-center justify-between py-2 text-sm"
              >
                <span>{profile.name}</span>
                {profile.active && <Badge variant="success">active</Badge>}
              </li>
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}
