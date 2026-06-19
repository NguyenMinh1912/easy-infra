import { AlertCircle, AtSign } from "lucide-react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useAsync } from "@/hooks/useAsync";
import { listIdentities } from "@/services/api";

import type { AwsServiceDetailProps } from "./types";

/**
 * SES detail page: the profile's verified email and domain identities, read
 * live from LocalStack. An unreachable endpoint comes back inside the response
 * envelope, so it renders as an expected outcome rather than a transport error.
 * Without a profile (no connection env) it explains how to connect.
 */
export function SesDetail({ service, profile }: AwsServiceDetailProps) {
  if (!profile) {
    return <NoConnection />;
  }
  return <SesIdentities profile={profile} service={service.id} />;
}

function SesIdentities({
  profile,
  service,
}: {
  profile: string;
  service: string;
}) {
  const state = useAsync(
    (signal) => listIdentities(profile, service, signal),
    [profile, service],
  );

  if (state.status === "loading") {
    return <Skeleton className="h-48 w-full" />;
  }
  if (state.status === "error" || state.data.error) {
    const message =
      state.status === "error" ? state.error.message : state.data.error;
    return (
      <Alert variant="destructive">
        <AlertCircle />
        <div>
          <AlertTitle>LocalStack unreachable</AlertTitle>
          <AlertDescription className="font-mono text-xs">
            {message}
          </AlertDescription>
        </div>
      </Alert>
    );
  }

  const identities = state.data.identities;
  if (identities.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Identities</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col items-center gap-3 py-8 text-center">
            <span className="flex size-10 items-center justify-center rounded-lg bg-muted">
              <AtSign className="size-5 text-muted-foreground" aria-hidden />
            </span>
            <p className="max-w-md text-sm text-muted-foreground">
              No identities yet. Verify an email or domain with the AWS CLI
              against this profile's endpoint and it will appear here.
            </p>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">
          Identities ({identities.length})
        </CardTitle>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Identity</TableHead>
              <TableHead>Type</TableHead>
              <TableHead className="text-right">Status</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {identities.map((id) => (
              <TableRow key={id.identity}>
                <TableCell className="font-mono">{id.identity}</TableCell>
                <TableCell>
                  <Badge variant="secondary">{identityTypeLabel(id.type)}</Badge>
                </TableCell>
                <TableCell className="text-right">
                  <Badge variant={id.verified ? "success" : "outline"}>
                    {id.verified ? "Verified" : "Pending"}
                  </Badge>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}

/** Friendly label for an SES identity type (e.g. EMAIL_ADDRESS → Email). */
function identityTypeLabel(type: string): string {
  switch (type) {
    case "EMAIL_ADDRESS":
      return "Email";
    case "DOMAIN":
    case "MANAGED_DOMAIN":
      return "Domain";
    default:
      return type;
  }
}

/** Shown when the page is not profile-scoped, so there is no endpoint to query. */
function NoConnection() {
  return (
    <Card>
      <CardContent className="py-8 text-center text-sm text-muted-foreground">
        Open this service from a profile to browse its SES identities.
      </CardContent>
    </Card>
  );
}
