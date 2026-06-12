import { AlertCircle } from "lucide-react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";

import { BackupsManager } from "./components/BackupsManager";
import { NoProject } from "./components/NoProject";
import { BackupsSkeleton } from "./components/BackupsSkeleton";
import { useBackups } from "./hooks/useBackups";

/**
 * Container for the backups screen: owns data loading via {@link useBackups}
 * and maps each async state onto a view. The only place wiring data to view.
 */
export function BackupsPage() {
  const { state, page, setPage, pageSize, reload } = useBackups();

  switch (state.status) {
    case "loading":
      return <BackupsSkeleton />;
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
        <BackupsManager
          data={state.data}
          page={page}
          pageSize={pageSize}
          setPage={setPage}
          reload={reload}
        />
      ) : (
        <NoProject />
      );
  }
}
