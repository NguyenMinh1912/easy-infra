import { AlertCircle } from "lucide-react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { useHashRoute } from "@/hooks/useHashRoute";
import { NoProject } from "./components/NoProject";
import { ServicesManager } from "./components/ServicesManager";
import { ServicesSkeleton } from "./components/ServicesSkeleton";
import { useServices } from "./hooks/useServices";

/** Service name deep-linked via `#/services/{name}`, or undefined. */
function focusedService(route: string): string | undefined {
  const match = route.match(/^\/services\/(.+)$/);
  return match ? decodeURIComponent(match[1]) : undefined;
}

/**
 * Container for the services screen: owns data loading via {@link useServices}
 * and maps each async state onto a view. The only place wiring data to view.
 */
export function ServicesPage() {
  const { state, reload } = useServices();
  const focus = focusedService(useHashRoute());

  switch (state.status) {
    case "loading":
      return <ServicesSkeleton />;
    case "error":
      return (
        <Alert variant="destructive">
          <AlertCircle />
          <div>
            <AlertTitle>Could not reach the API</AlertTitle>
            <AlertDescription>
              {state.error.message}. Make sure <code>easy-infra serve</code> is
              running, then refresh.
            </AlertDescription>
          </div>
        </Alert>
      );
    case "success":
      if (!state.data.initialized) {
        return <NoProject />;
      }
      if (!state.data.activeProfile) {
        return (
          <Alert>
            <AlertCircle />
            <div>
              <AlertTitle>No active profile</AlertTitle>
              <AlertDescription>
                Services belong to a profile. Activate a profile from the
                sidebar to manage its services.
              </AlertDescription>
            </div>
          </Alert>
        );
      }
      return (
        <ServicesManager data={state.data} reload={reload} focusService={focus} />
      );
  }
}
