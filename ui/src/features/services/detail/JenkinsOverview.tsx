import { AlertCircle, ChevronRight, Hammer, Loader2, Play, RotateCw } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { toast } from "sonner";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { getJenkinsInfo, listJobs, triggerBuild } from "@/services/api";
import type { JenkinsInfo, JobInfo } from "@/types/jenkins";

import { DefinitionSummary } from "./DefinitionSummary";
import { BuildList } from "./jenkins/BuildList";
import { jobStatusFor } from "./jenkins/jobStatus";
import type { OverviewProps } from "./types";

/** How often to re-poll Jenkins so building states update, in milliseconds. */
const POLL_INTERVAL = 5000;

type State =
  | { status: "loading" }
  | { status: "connected"; info: JenkinsInfo; jobs: JobInfo[] }
  | { status: "unreachable"; error: string };

/**
 * Jenkins-specific overview. On a profile-scoped page it shows the live server:
 * an instance card (version, mode, job count) and a jobs table read from the
 * Jenkins REST API (polled), where opening a job reveals its recent builds.
 * Without a profile (no connection env) it falls back to a config summary,
 * mirroring the Redis overview.
 */
export function JenkinsOverview({ service, profile }: OverviewProps) {
  if (!profile) {
    return <JenkinsSummary service={service} />;
  }
  return <JenkinsLive service={service} profile={profile} />;
}

/** The live, profile-scoped view: instance card + jobs table. */
function JenkinsLive({
  service,
  profile,
}: {
  service: OverviewProps["service"];
  profile: string;
}) {
  const { state, retry } = useJenkins(profile, service.id);
  const [openJob, setOpenJob] = useState<string | null>(null);

  const version =
    (state.status === "connected" && state.info.version) ||
    stringField(service.config.version) ||
    "—";

  return (
    <div className="mx-auto w-full max-w-4xl space-y-6">
      <header className="flex items-center gap-2">
        <Hammer className="size-5 text-muted-foreground" aria-hidden />
        <h1 className="text-lg font-semibold">Jenkins</h1>
        <Badge variant="secondary">v{version}</Badge>
      </header>

      {state.status === "loading" && <Skeleton className="h-48 w-full" />}

      {state.status === "unreachable" && (
        <Alert variant="destructive">
          <AlertCircle />
          <div className="flex-1">
            <AlertTitle>Jenkins isn't responding</AlertTitle>
            <AlertDescription className="font-mono text-xs">
              {state.error}
            </AlertDescription>
            <Button variant="outline" size="sm" className="mt-3" onClick={retry}>
              <RotateCw aria-hidden />
              Retry
            </Button>
          </div>
        </Alert>
      )}

      {state.status === "connected" && (
        <>
          <InstanceCard info={state.info} />
          <JobsTable
            jobs={state.jobs}
            openJob={openJob}
            onToggle={(name) =>
              setOpenJob((prev) => (prev === name ? null : name))
            }
            onTriggered={retry}
            profile={profile}
            service={service.id}
          />
        </>
      )}
    </div>
  );
}

/** Summary card of the running instance. */
function InstanceCard({ info }: { info: JenkinsInfo }) {
  const rows: [string, string][] = [
    ["Version", info.version || "—"],
    ["Node", info.nodeName || "built-in"],
    ["Mode", info.mode || "—"],
    ["Jobs", String(info.jobCount)],
  ];
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Instance</CardTitle>
        {info.description && (
          <p className="text-sm text-muted-foreground">{info.description}</p>
        )}
      </CardHeader>
      <CardContent>
        <dl className="grid grid-cols-2 gap-x-6 gap-y-3 sm:grid-cols-4">
          {rows.map(([label, value]) => (
            <div key={label}>
              <dt className="text-xs text-muted-foreground">{label}</dt>
              <dd className="text-sm font-medium text-foreground">{value}</dd>
            </div>
          ))}
        </dl>
        {info.quietingDown && (
          <p className="mt-3 text-sm text-amber-600 dark:text-amber-500">
            Server is quieting down for shutdown.
          </p>
        )}
      </CardContent>
    </Card>
  );
}

/** The jobs list; each row toggles an inline build-history panel. */
function JobsTable({
  jobs,
  openJob,
  onToggle,
  onTriggered,
  profile,
  service,
}: {
  jobs: JobInfo[];
  openJob: string | null;
  onToggle: (name: string) => void;
  onTriggered: () => void;
  profile: string;
  service: string;
}) {
  // Jobs with an in-flight trigger request, so the button shows a spinner and
  // can't be double-clicked while the request is pending.
  const [pending, setPending] = useState<Set<string>>(new Set());

  const build = async (job: string) => {
    setPending((prev) => new Set(prev).add(job));
    try {
      await triggerBuild(profile, service, job);
      toast.success(`Build of "${job}" started`);
      onTriggered();
    } catch (cause) {
      toast.error(`Couldn't start build of "${job}"`, {
        description: cause instanceof Error ? cause.message : String(cause),
      });
    } finally {
      setPending((prev) => {
        const next = new Set(prev);
        next.delete(job);
        return next;
      });
    }
  };

  if (jobs.length === 0) {
    return (
      <Card>
        <CardContent className="py-8 text-center text-sm text-muted-foreground">
          This Jenkins server has no jobs.
        </CardContent>
      </Card>
    );
  }
  return (
    <div className="space-y-2">
      {jobs.map((job) => {
        const status = jobStatusFor(job.color);
        const open = openJob === job.name;
        const building = pending.has(job.name);
        return (
          <Card key={job.name} className="overflow-hidden">
            <div className="flex items-center gap-3 pr-3">
              <button
                type="button"
                onClick={() => onToggle(job.name)}
                aria-expanded={open}
                className="flex flex-1 items-center gap-3 p-4 text-left transition-colors hover:bg-accent focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none"
              >
                <span
                  className={`size-2.5 shrink-0 rounded-full ${status.dotClass} ${
                    status.building ? "motion-safe:animate-pulse" : ""
                  }`}
                  aria-hidden
                />
                <div className="min-w-0 flex-1">
                  <span className="font-medium text-foreground">{job.name}</span>
                </div>
                <span className="text-xs text-muted-foreground">{status.label}</span>
                {job.lastBuild ? (
                  <span className="font-mono text-xs text-muted-foreground">
                    #{job.lastBuild}
                  </span>
                ) : null}
                <ChevronRight
                  className={`size-4 shrink-0 text-muted-foreground transition-transform ${
                    open ? "rotate-90" : ""
                  }`}
                  aria-hidden
                />
              </button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                disabled={building}
                onClick={() => void build(job.name)}
                aria-label={`Build ${job.name} now`}
              >
                {building ? (
                  <Loader2 className="animate-spin" aria-hidden />
                ) : (
                  <Play aria-hidden />
                )}
                Build now
              </Button>
            </div>
            {open && (
              <div className="border-t px-4 py-3">
                <BuildList profile={profile} service={service} job={job.name} />
              </div>
            )}
          </Card>
        );
      })}
    </div>
  );
}

/** The config summary shown when the page isn't profile-scoped. */
function JenkinsSummary({ service }: Pick<OverviewProps, "service">) {
  const version = String(service.config.version ?? "—");
  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <div className="flex items-center gap-3">
            <span className="flex size-10 items-center justify-center rounded-lg bg-muted">
              <Hammer className="size-5 text-muted-foreground" aria-hidden />
            </span>
            <div className="flex-1">
              <CardTitle className="text-base">Jenkins</CardTitle>
              <p className="text-sm text-muted-foreground">
                CI/CD automation server
              </p>
            </div>
            <Badge variant="secondary">v{version}</Badge>
          </div>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground">
            This profile owns this service. Its config — image version plus the
            connection details (host, port, optional credentials) — is edited
            under{" "}
            <span className="font-medium text-foreground">
              Profiles → Settings
            </span>
            .
          </p>
        </CardContent>
      </Card>
      <DefinitionSummary config={service.config} />
    </div>
  );
}

/**
 * Poll the Jenkins instance info and jobs for the profile/service. The first
 * fetch shows `loading`; later polls update in place so building states refresh
 * without flicker. Keeps polling even while unreachable so the page recovers on
 * its own; `retry` forces an immediate refetch.
 */
function useJenkins(profile: string, service: string) {
  const [state, setState] = useState<State>({ status: "loading" });
  const [nonce, setNonce] = useState(0);

  useEffect(() => {
    let cancelled = false;
    let timer: ReturnType<typeof setTimeout> | null = null;
    const controller = new AbortController();
    setState({ status: "loading" });

    const poll = async () => {
      try {
        const [info, jobs] = await Promise.all([
          getJenkinsInfo(profile, service, controller.signal),
          listJobs(profile, service, controller.signal),
        ]);
        if (cancelled) return;
        const error = info.error || jobs.error;
        setState(
          error
            ? { status: "unreachable", error }
            : {
                status: "connected",
                info: { quietingDown: false, jobCount: 0, ...info },
                jobs: jobs.jobs,
              },
        );
      } catch (cause) {
        if (cancelled || controller.signal.aborted) return;
        setState({
          status: "unreachable",
          error: cause instanceof Error ? cause.message : String(cause),
        });
      }
      if (!cancelled) timer = setTimeout(() => void poll(), POLL_INTERVAL);
    };
    void poll();

    return () => {
      cancelled = true;
      controller.abort();
      if (timer) clearTimeout(timer);
    };
  }, [profile, service, nonce]);

  const retry = useCallback(() => setNonce((n) => n + 1), []);
  return { state, retry };
}

/** Read a config value as a non-empty trimmed string, or undefined. */
function stringField(value: unknown): string | undefined {
  if (value === undefined || value === null) return undefined;
  const s = String(value).trim();
  return s === "" ? undefined : s;
}
