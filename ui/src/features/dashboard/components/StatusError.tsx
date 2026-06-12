import { AlertCircle } from "lucide-react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";

interface StatusErrorProps {
  message: string;
}

/** Error state shown when the status request fails. */
export function StatusError({ message }: StatusErrorProps) {
  return (
    <Alert variant="destructive">
      <AlertCircle />
      <div>
        <AlertTitle>Could not reach the API</AlertTitle>
        <AlertDescription>
          {message}. Make sure <code>easy-infra serve</code> is running, then
          refresh.
        </AlertDescription>
      </div>
    </Alert>
  );
}
