import type { Status } from "@/types/status";
import { ActiveProfileCard } from "./ActiveProfileCard";
import { EmptyProject } from "./EmptyProject";
import { ProfilesCard } from "./ProfilesCard";
import { ServicesCard } from "./ServicesCard";

interface DashboardProps {
  status: Status;
}

/**
 * Renders the dashboard for a loaded {@link Status}. Pure presentation: it
 * receives data and composes the cards, with no knowledge of how the data was
 * fetched.
 */
export function Dashboard({ status }: DashboardProps) {
  if (!status.initialized) {
    return <EmptyProject />;
  }

  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <ActiveProfileCard activeProfile={status.activeProfile} />
      <ProfilesCard profiles={status.profiles} />
      <ServicesCard services={status.services} />
    </div>
  );
}
