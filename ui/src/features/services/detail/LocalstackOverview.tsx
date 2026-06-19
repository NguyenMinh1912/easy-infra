import { ArrowLeft, ChevronRight, Cloud } from "lucide-react";
import { useState } from "react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { ServiceConfig } from "@/types/service";

import { DefinitionSummary } from "./DefinitionSummary";
import { AWS_SERVICES, awsServiceFor } from "./localstack/aws-services";
import type { OverviewProps } from "./types";

/**
 * LocalStack-specific overview. LocalStack emulates a bundle of AWS services, so
 * its detail page is a launcher: a grid of the AWS service apps easy-infra
 * supports (see {@link AWS_SERVICES}), each flagged with whether this profile
 * enables it. Opening a card drills into that AWS service's own detail page —
 * an in-page navigation with a back link, like the MinIO object browser.
 */
export function LocalstackOverview({ service, profile }: OverviewProps) {
  const [selected, setSelected] = useState<string | null>(null);

  const enabled = enabledAwsServices(service.config);
  const app = selected ? awsServiceFor(selected) : undefined;

  if (app) {
    const Detail = app.Detail;
    const Icon = app.icon;
    return (
      <div className="space-y-6">
        <Button
          variant="ghost"
          size="sm"
          className="-ml-2 gap-1.5 text-muted-foreground"
          onClick={() => setSelected(null)}
        >
          <ArrowLeft className="size-4" aria-hidden />
          LocalStack
        </Button>
        <Card>
          <CardHeader>
            <div className="flex items-center gap-3">
              <span className="flex size-10 items-center justify-center rounded-lg bg-muted">
                <Icon className="size-5 text-muted-foreground" aria-hidden />
              </span>
              <div className="flex-1">
                <CardTitle className="text-base">{app.name}</CardTitle>
                <p className="text-sm text-muted-foreground">{app.blurb}</p>
              </div>
              {enabled.has(app.id) ? (
                <Badge variant="secondary">Enabled</Badge>
              ) : (
                <Badge variant="outline">Not in config</Badge>
              )}
            </div>
          </CardHeader>
        </Card>
        <Detail service={service} profile={profile} />
      </div>
    );
  }

  const version = String(service.config.version ?? "—");

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <div className="flex items-center gap-3">
            <span className="flex size-10 items-center justify-center rounded-lg bg-muted">
              <Cloud className="size-5 text-muted-foreground" aria-hidden />
            </span>
            <div className="flex-1">
              <CardTitle className="text-base">LocalStack</CardTitle>
              <p className="text-sm text-muted-foreground">AWS cloud emulator</p>
            </div>
            <Badge variant="secondary">v{version}</Badge>
          </div>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">
            Open an AWS service to see its detail page. The services this profile
            emulates are configured under{" "}
            <span className="font-medium text-foreground">
              Profiles → Settings
            </span>
            .
          </p>
        </CardContent>
      </Card>

      <div className="grid gap-3 sm:grid-cols-2">
        {AWS_SERVICES.map((aws) => {
          const Icon = aws.icon;
          return (
            <button
              key={aws.id}
              type="button"
              onClick={() => setSelected(aws.id)}
              className="flex items-center gap-3 rounded-lg border bg-card p-4 text-left transition-colors hover:bg-accent focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none"
            >
              <span className="flex size-10 items-center justify-center rounded-lg bg-muted">
                <Icon className="size-5 text-muted-foreground" aria-hidden />
              </span>
              <div className="flex-1">
                <div className="flex items-center gap-2">
                  <span className="font-medium">{aws.label}</span>
                  {enabled.has(aws.id) && (
                    <Badge variant="secondary" className="text-xs">
                      Enabled
                    </Badge>
                  )}
                </div>
                <p className="text-sm text-muted-foreground">{aws.blurb}</p>
              </div>
              <ChevronRight
                className="size-4 text-muted-foreground"
                aria-hidden
              />
            </button>
          );
        })}
      </div>

      <DefinitionSummary config={service.config} />
    </div>
  );
}

/**
 * The set of AWS service ids enabled in a LocalStack config. The `services`
 * field is a comma-separated string (e.g. `s3,sqs,sns`); normalise it the way
 * the backend does — trimming whitespace, lower-casing, dropping blanks. Also
 * tolerates a list form in case a config arrives pre-split.
 */
function enabledAwsServices(config: ServiceConfig): Set<string> {
  const raw = config.services;
  const parts =
    typeof raw === "string"
      ? raw.split(",")
      : Array.isArray(raw)
        ? raw.filter((s): s is string => typeof s === "string")
        : [];
  const enabled = new Set<string>();
  for (const part of parts) {
    const id = part.trim().toLowerCase();
    if (id !== "") enabled.add(id);
  }
  return enabled;
}
