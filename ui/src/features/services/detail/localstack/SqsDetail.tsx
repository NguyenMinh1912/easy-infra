import { AlertCircle, Inbox } from "lucide-react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
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
import { listQueues } from "@/services/api";

import type { AwsServiceDetailProps } from "./types";

/**
 * SQS detail page: the profile's queues with their message counts, read live
 * from LocalStack. An unreachable endpoint comes back inside the response
 * envelope, so it renders as an expected outcome rather than a transport error.
 * Without a profile (no connection env) it explains how to connect.
 */
export function SqsDetail({ service, profile, region }: AwsServiceDetailProps) {
  if (!profile) {
    return <NoConnection />;
  }
  return <SqsQueues profile={profile} service={service.id} region={region} />;
}

function SqsQueues({
  profile,
  service,
  region,
}: {
  profile: string;
  service: string;
  region?: string;
}) {
  const state = useAsync(
    (signal) => listQueues(profile, service, region, signal),
    [profile, service, region],
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

  const queues = state.data.queues;
  if (queues.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Queues</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col items-center gap-3 py-8 text-center">
            <span className="flex size-10 items-center justify-center rounded-lg bg-muted">
              <Inbox className="size-5 text-muted-foreground" aria-hidden />
            </span>
            <p className="max-w-md text-sm text-muted-foreground">
              No queues yet. Create one with the AWS CLI against this profile's
              endpoint and it will appear here.
            </p>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Queues ({queues.length})</CardTitle>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead className="text-right">Messages</TableHead>
              <TableHead className="text-right">In flight</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {queues.map((q) => (
              <TableRow key={q.url}>
                <TableCell className="font-mono">{q.name}</TableCell>
                <TableCell className="text-right tabular-nums">
                  {q.messages}
                </TableCell>
                <TableCell className="text-right tabular-nums">
                  {q.inFlight}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}

/** Shown when the page is not profile-scoped, so there is no endpoint to query. */
function NoConnection() {
  return (
    <Card>
      <CardContent className="py-8 text-center text-sm text-muted-foreground">
        Open this service from a profile to browse its SQS queues.
      </CardContent>
    </Card>
  );
}
