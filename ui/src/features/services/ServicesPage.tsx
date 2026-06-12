import { AlertCircle } from "lucide-react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { NoProject } from "./components/NoProject";
import { ServicesManager } from "./components/ServicesManager";
import { ServicesSkeleton } from "./components/ServicesSkeleton";
import { useServices } from "./hooks/useServices";

/**
 * Container for the services screen: owns data loading via {@link useServices}
 * and maps each async state onto a view. The only place wiring data to view.
 */
export function ServicesPage() {
  const { state, reload } = useServices();

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
      return state.data.initialized ? (
        <ServicesManager data={state.data} reload={reload} />
      ) : (
        <NoProject />
      );
  }
}
