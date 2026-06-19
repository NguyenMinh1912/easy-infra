import { Inbox } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

import type { AwsServiceDetailProps } from "./types";

/**
 * SQS detail page. LocalStack provisioning is not live yet (the provider
 * reports `notImplemented`), so the queue list shows an empty state describing
 * what appears once the service is running. The body sits under the shared
 * header the overview renders from the app catalog.
 */
export function SqsDetail(_: AwsServiceDetailProps) {
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
            Queues appear here once LocalStack is running with the{" "}
            <span className="font-mono text-foreground">sqs</span> service
            enabled. Apply this profile to start it.
          </p>
        </div>
      </CardContent>
    </Card>
  );
}
