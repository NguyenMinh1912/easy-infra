import { AlertCircle } from "lucide-react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";

interface ProfilesErrorProps {
  message: string;
}

/** Error state shown when the profiles request fails. */
export function ProfilesError({ message }: ProfilesErrorProps) {
  return (
    <Alert variant="destructive">
      <AlertCircle />
      <div>
        <AlertTitle>Could not load profiles</AlertTitle>
        <AlertDescription>
          {message}. Make sure <code>easy-infra serve</code> is running, then
          refresh.
        </AlertDescription>
      </div>
    </Alert>
  );
}
