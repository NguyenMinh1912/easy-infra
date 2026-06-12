import { AlertCircle, Boxes } from "lucide-react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";

import { NoProject } from "../components/NoProject";
import { ServicesSkeleton } from "../components/ServicesSkeleton";
import { useServices } from "../hooks/useServices";
import { overviewFor } from "./overview-registry";
import { ServiceActions } from "./ServiceActions";
import { ServiceDetailLayout } from "./ServiceDetailLayout";

interface ServiceDetailPageProps {
  /** Service to show, taken from the `#/services/{name}` route. */
  name: string;
  /** Profile scoping the page, when reached via `#/profiles/{p}/services/{name}`. */
  profile?: string;
}

/**
 * Container for a single service's detail screen: owns data loading via
 * {@link useServices}, maps each async state onto a view, and renders the
 * service's profile info inside the shared {@link ServiceDetailLayout}. The
 * navbar carries the service action menu; the overview is service-specific
 * (postgres ships its own).
 */
export function ServiceDetailPage({ name, profile }: ServiceDetailPageProps) {
  const { state } = useServices();

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
    case "success": {
      if (!state.data.initialized) {
        return <NoProject />;
      }
      const service = state.data.services.find((s) => s.name === name);
      if (!service) {
        return <ServiceNotFound name={name} />;
      }

      const Overview = overviewFor(service.name);

      return (
        <ServiceDetailLayout
          key={service.name}
          actions={<ServiceActions service={service} />}
        >
          <Overview service={service} profile={profile} />
        </ServiceDetailLayout>
      );
    }
  }
}

/** Shown when the route names a service the project does not define. */
function ServiceNotFound({ name }: { name: string }) {
  return (
    <Card className="flex flex-col items-center gap-3 p-10 text-center">
      <span className="flex size-10 items-center justify-center rounded-lg bg-muted">
        <Boxes className="size-5 text-muted-foreground" aria-hidden />
      </span>
      <div className="space-y-1">
        <p className="font-medium">
          No service named <span className="font-mono">{name}</span>
        </p>
        <p className="text-sm text-muted-foreground">
          It is not defined in this project.
        </p>
      </div>
      <Button asChild size="sm" variant="outline">
        <a href="#/services">Back to services</a>
      </Button>
    </Card>
  );
}
