import type { ProfilesResult } from "@/types/profiles";

import type { ProfileActions } from "../hooks/useProfiles";
import { CreateProfileForm } from "./CreateProfileForm";
import { ProfileList } from "./ProfileList";

interface ProfilesProps {
  data: ProfilesResult;
  actions: ProfileActions;
}

/**
 * Renders the profiles screen for loaded data. Composes the create form and the
 * profile list; it receives data and actions and holds no fetching logic.
 */
export function Profiles({ data, actions }: ProfilesProps) {
  return (
    <div className="space-y-6">
      <CreateProfileForm
        existingNames={data.profiles.map((profile) => profile.name)}
        onCreate={actions.create}
      />
      <ProfileList
        profiles={data.profiles}
        onActivate={actions.activate}
        onRemove={actions.remove}
      />
    </div>
  );
}
