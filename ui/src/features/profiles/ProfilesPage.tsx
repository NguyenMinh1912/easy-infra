import { Profiles } from "./components/Profiles";
import { ProfilesError } from "./components/ProfilesError";
import { ProfilesSkeleton } from "./components/ProfilesSkeleton";
import { useProfiles } from "./hooks/useProfiles";

/**
 * Container for the profiles screen: owns data loading and the CRUD actions via
 * {@link useProfiles} and maps each async state onto a presentational
 * component. This is the only place in the feature that wires data to view.
 */
export function ProfilesPage() {
  const { state, actions } = useProfiles();

  switch (state.status) {
    case "loading":
      return <ProfilesSkeleton />;
    case "error":
      return <ProfilesError message={state.error.message} />;
    case "success":
      return <Profiles data={state.data} actions={actions} />;
  }
}
