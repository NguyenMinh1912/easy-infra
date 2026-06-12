import { ProfileSettings } from "./components/ProfileSettings";
import { ProfilesError } from "./components/ProfilesError";
import { ProfilesSkeleton } from "./components/ProfilesSkeleton";
import { useProfileConfig } from "./hooks/useProfileConfig";

interface ProfileSettingsPageProps {
  /** Profile being configured, taken from the route. */
  name: string;
}

/**
 * Container for a single profile's settings: owns data loading and the save
 * action via {@link useProfileConfig} and maps each async state onto a view.
 * Keyed by profile name so its draft resets when navigating between profiles.
 */
export function ProfileSettingsPage({ name }: ProfileSettingsPageProps) {
  const { state, actions } = useProfileConfig(name);

  switch (state.status) {
    case "loading":
      return <ProfilesSkeleton />;
    case "error":
      return <ProfilesError message={state.error.message} />;
    case "success":
      return (
        <ProfileSettings key={name} data={state.data} actions={actions} />
      );
  }
}
