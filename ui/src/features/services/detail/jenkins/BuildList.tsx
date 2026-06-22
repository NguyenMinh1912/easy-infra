import { AlertCircle, ScrollText } from "lucide-react";
import { useEffect, useState } from "react";

import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { listBuilds } from "@/services/api";
import type { BuildInfo } from "@/types/jenkins";

import { BuildLogDialog } from "./BuildLogDialog";
import { buildResultLabel } from "./jobStatus";

type State =
  | { status: "loading" }
  | { status: "loaded"; builds: BuildInfo[] }
  | { status: "error"; error: string };

/** Recent builds of one job, fetched on open and when the job changes. */
export function BuildList({
  profile,
  service,
  job,
}: {
  profile: string;
  service: string;
  job: string;
}) {
  const [state, setState] = useState<State>({ status: "loading" });
  const [logBuild, setLogBuild] = useState<BuildInfo | null>(null);

  useEffect(() => {
    let cancelled = false;
    const controller = new AbortController();
    setState({ status: "loading" });

    listBuilds(profile, service, job, controller.signal)
      .then((res) => {
        if (cancelled) return;
        setState(
          res.error
            ? { status: "error", error: res.error }
            : { status: "loaded", builds: res.builds },
        );
      })
      .catch((cause) => {
        if (cancelled || controller.signal.aborted) return;
        setState({
          status: "error",
          error: cause instanceof Error ? cause.message : String(cause),
        });
      });

    return () => {
      cancelled = true;
      controller.abort();
    };
  }, [profile, service, job]);

  if (state.status === "loading") {
    return <Skeleton className="h-24 w-full" />;
  }
  if (state.status === "error") {
    return (
      <Alert variant="destructive">
        <AlertCircle />
        <AlertDescription className="font-mono text-xs">
          {state.error}
        </AlertDescription>
      </Alert>
    );
  }
  if (state.builds.length === 0) {
    return (
      <p className="py-4 text-center text-sm text-muted-foreground">
        No builds yet.
      </p>
    );
  }

  return (
    <>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Build</TableHead>
            <TableHead>Result</TableHead>
            <TableHead>When</TableHead>
            <TableHead>Duration</TableHead>
            <TableHead className="w-0" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {state.builds.map((build) => (
            <TableRow key={build.number}>
              <TableCell className="font-mono">#{build.number}</TableCell>
              <TableCell>{buildResultLabel(build.result, build.building)}</TableCell>
              <TableCell className="text-muted-foreground">
                {build.timestamp
                  ? new Date(build.timestamp).toLocaleString()
                  : "—"}
              </TableCell>
              <TableCell className="text-muted-foreground">
                {build.building ? "running…" : formatDuration(build.duration)}
              </TableCell>
              <TableCell className="text-right">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setLogBuild(build)}
                  aria-label={`View log of build #${build.number}`}
                >
                  <ScrollText aria-hidden />
                  Log
                </Button>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
      <BuildLogDialog
        profile={profile}
        service={service}
        job={job}
        build={logBuild}
        onOpenChange={(open) => {
          if (!open) setLogBuild(null);
        }}
      />
    </>
  );
}

/** Render a millisecond duration compactly (e.g. "1m 12s", "850ms"). */
function formatDuration(ms: number): string {
  if (ms <= 0) return "—";
  if (ms < 1000) return `${ms}ms`;
  const seconds = Math.round(ms / 1000);
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  const rest = seconds % 60;
  return rest ? `${minutes}m ${rest}s` : `${minutes}m`;
}
