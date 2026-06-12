import { useState } from "react";
import { CheckCircle2, Loader2, PlugZap, XCircle } from "lucide-react";

import { Button } from "@/components/ui/button";
import { checkServiceConnection } from "@/services/api";

import {
  ProfileServiceConfigCard,
  type ConfigRow,
  type ProfileServiceConfigCardProps,
} from "./ProfileServiceConfigCard";

type CheckState =
  | { status: "idle" }
  | { status: "checking" }
  | { status: "ok" }
  | { status: "error"; message: string };

/**
 * Profile config editor for minio. Reuses the generic key/value editor and adds
 * a "Health check" button that probes the endpoint with the current (unsaved)
 * config, mirroring postgres's connection check. The backend `/check` endpoint
 * calls the service's `Health`, so this stays a thin UI affordance.
 */
export function MinioProfileConfigCard({
  onChange,
  ...props
}: ProfileServiceConfigCardProps) {
  const { name, profileName, rows, disabled } = props;
  const [check, setCheck] = useState<CheckState>({ status: "idle" });

  // Any edit invalidates a previous result, so reset to idle on change.
  const handleChange = (next: ConfigRow[]) => {
    setCheck({ status: "idle" });
    onChange(next);
  };

  const runCheck = async () => {
    setCheck({ status: "checking" });
    try {
      const result = await checkServiceConnection(
        profileName,
        name,
        rowsToConfig(rows),
      );
      setCheck(
        result.ok
          ? { status: "ok" }
          : { status: "error", message: result.error ?? "health check failed" },
      );
    } catch (cause) {
      setCheck({
        status: "error",
        message: cause instanceof Error ? cause.message : String(cause),
      });
    }
  };

  const busy = disabled || check.status === "checking";

  const footer = (
    <div className="flex items-center gap-3 pt-1">
      <Button
        type="button"
        variant="outline"
        size="sm"
        disabled={busy}
        onClick={runCheck}
      >
        {check.status === "checking" ? (
          <Loader2 className="animate-spin" aria-hidden />
        ) : (
          <PlugZap aria-hidden />
        )}
        Health check
      </Button>
      {check.status === "ok" && (
        <span className="flex items-center gap-1.5 text-sm text-emerald-600 dark:text-emerald-500">
          <CheckCircle2 className="size-4" aria-hidden />
          Healthy
        </span>
      )}
      {check.status === "error" && (
        <span className="flex items-center gap-1.5 text-sm text-destructive">
          <XCircle className="size-4 shrink-0" aria-hidden />
          {check.message}
        </span>
      )}
    </div>
  );

  return (
    <ProfileServiceConfigCard {...props} onChange={handleChange} footer={footer} />
  );
}

/** Build the env config object posted to the check from the draft rows. */
function rowsToConfig(rows: ConfigRow[]): Record<string, unknown> {
  const config: Record<string, unknown> = {};
  for (const { key, value } of rows) {
    const trimmed = key.trim();
    if (trimmed) config[trimmed] = value;
  }
  return config;
}
