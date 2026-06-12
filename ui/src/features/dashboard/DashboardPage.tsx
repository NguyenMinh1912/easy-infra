import { Dashboard } from "./components/Dashboard";
import { DashboardSkeleton } from "./components/DashboardSkeleton";
import { StatusError } from "./components/StatusError";
import { useStatus } from "./hooks/useStatus";

/**
 * Container for the dashboard screen: owns data loading via {@link useStatus}
 * and maps each async state onto a presentational component. This is the only
 * place in the feature that wires data to view.
 */
export function DashboardPage() {
  const state = useStatus();

  switch (state.status) {
    case "loading":
      return <DashboardSkeleton />;
    case "error":
      return <StatusError message={state.error.message} />;
    case "success":
      return <Dashboard status={state.data} />;
  }
}
