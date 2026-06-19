import { AlertCircle, Play } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { executeQuery } from "@/services/api";
import type { QueryResult } from "@/types/console";

import { QueryResultTable } from "../console/QueryResultTable";

interface RedisConsoleProps {
  /** Profile whose saved connection config the command runs against. */
  profile: string;
  /** Service name within the profile (the API path segment). */
  service: string;
}

/** State of the current (or last) command execution. */
type RunState =
  | { status: "idle" }
  | { status: "running" }
  | { status: "done"; result: QueryResult }
  | { status: "failed"; error: string };

/**
 * redis-cli-style console against one profile's Redis: type a command, run it,
 * and see the reply rendered as a result table (one row per array element).
 * Reuses the console's /query endpoint and result table. Command failures come
 * back inside the response envelope, so they render as an expected outcome
 * rather than a transport error.
 */
export function RedisConsole({ profile, service }: RedisConsoleProps) {
  const [command, setCommand] = useState("");
  const [run, setRun] = useState<RunState>({ status: "idle" });
  // Recent commands, most-recent first, for quick re-runs.
  const [history, setHistory] = useState<string[]>([]);

  const controllerRef = useRef<AbortController | null>(null);
  useEffect(() => () => controllerRef.current?.abort(), []);

  const runCommand = useCallback(
    (raw: string) => {
      const cmd = raw.trim();
      if (!cmd) return;
      controllerRef.current?.abort();
      const controller = new AbortController();
      controllerRef.current = controller;
      setRun({ status: "running" });
      setHistory((prev) => [cmd, ...prev.filter((c) => c !== cmd)].slice(0, 10));

      executeQuery(profile, service, cmd, controller.signal)
        .then((result) => {
          if (controller.signal.aborted) return;
          if (result.error) {
            setRun({ status: "failed", error: result.error });
          } else {
            setRun({ status: "done", result });
          }
        })
        .catch((cause: unknown) => {
          if (controller.signal.aborted) return;
          setRun({
            status: "failed",
            error: cause instanceof Error ? cause.message : String(cause),
          });
        });
    },
    [profile, service],
  );

  const running = run.status === "running";

  return (
    <div className="space-y-4">
      <form
        className="flex items-center gap-2"
        onSubmit={(e) => {
          e.preventDefault();
          runCommand(command);
        }}
      >
        <Input
          value={command}
          onChange={(e) => setCommand(e.target.value)}
          placeholder="Command, e.g. GET session:123"
          aria-label="Redis command"
          className="min-w-0 flex-1 font-mono"
          autoComplete="off"
          spellCheck={false}
        />
        <Button type="submit" size="sm" disabled={running || command.trim() === ""}>
          <Play aria-hidden /> Run
        </Button>
      </form>

      {history.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {history.map((cmd) => (
            <button
              key={cmd}
              type="button"
              onClick={() => {
                setCommand(cmd);
                runCommand(cmd);
              }}
              className="rounded border border-border px-2 py-0.5 font-mono text-xs text-muted-foreground hover:bg-muted hover:text-foreground"
            >
              {cmd}
            </button>
          ))}
        </div>
      )}

      {run.status === "running" && (
        <div className="space-y-2" aria-label="Running command">
          <Skeleton className="h-8 w-full" />
          <Skeleton className="h-8 w-2/3" />
        </div>
      )}
      {run.status === "failed" && (
        <Alert variant="destructive">
          <AlertCircle />
          <div>
            <AlertTitle>Command failed</AlertTitle>
            <AlertDescription className="font-mono text-xs">
              {run.error}
            </AlertDescription>
          </div>
        </Alert>
      )}
      {run.status === "done" && <QueryResultTable result={run.result} />}
    </div>
  );
}
