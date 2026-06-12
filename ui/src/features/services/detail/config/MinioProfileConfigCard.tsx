import { useMemo, useState } from "react";
import { CheckCircle2, Loader2, PlugZap, XCircle } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { checkServiceConnection } from "@/services/api";

import type {
  ConfigRow,
  ProfileServiceConfigCardProps,
} from "./ProfileServiceConfigCard";

/** A named text/number field surfaced by the form. */
interface TextField {
  key: "host" | "port" | "consolePort" | "user" | "password";
  label: string;
  placeholder: string;
  type?: string;
}

/** Text/number fields this card edits, in display order. */
const TEXT_FIELDS: readonly TextField[] = [
  { key: "host", label: "Host", placeholder: "localhost" },
  { key: "port", label: "Port", placeholder: "9000", type: "number" },
  { key: "consolePort", label: "Console port", placeholder: "9001", type: "number" },
  { key: "user", label: "User", placeholder: "minioadmin" },
  { key: "password", label: "Password", placeholder: "", type: "password" },
];

/** Keys this card owns; rows with any other key are preserved untouched. */
const KNOWN_KEYS = new Set<string>([
  ...TEXT_FIELDS.map((f) => f.key),
  "secure",
  "version",
  "buckets",
]);

/** The editable values this card surfaces, all held as strings/booleans. */
interface MinioFields {
  host: string;
  port: string;
  consolePort: string;
  user: string;
  password: string;
  version: string;
  buckets: string;
  secure: boolean;
}

type CheckState =
  | { status: "idle" }
  | { status: "checking" }
  | { status: "ok" }
  | { status: "error"; message: string };

/**
 * Tailored profile config editor for minio. Instead of the generic editable
 * key/value rows, it shows a form with named fields — connection details,
 * image version and the buckets to create — so a user fills in information
 * without having to know (or be able to mistype) the underlying config keys.
 * Unknown keys are preserved untouched, mirroring the postgres card and the
 * "don't special-case service names" convention.
 *
 * Like the generic card it owns no persisted state — the parent holds the rows
 * draft — so it projects the fields back onto rows on every change. The backend
 * coerces these string values (numeric ports, comma-separated buckets, a
 * "true"/"false" secure flag), so the rows stay a plain string map.
 */
export function MinioProfileConfigCard({
  name,
  profileName,
  rows,
  onChange,
  disabled,
}: ProfileServiceConfigCardProps) {
  const fields = useMemo(() => readState(rows), [rows]);
  const [check, setCheck] = useState<CheckState>({ status: "idle" });

  // Any edit invalidates a previous health-check result, so reset to idle.
  const setField = <K extends keyof MinioFields>(key: K, value: MinioFields[K]) => {
    setCheck({ status: "idle" });
    onChange(fieldsToRows({ ...fields, [key]: value }, rows));
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

  return (
    <Card>
      <CardHeader>
        <CardTitle className="font-mono text-base">{name}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-3 sm:grid-cols-2">
          {TEXT_FIELDS.map((field) => (
            <div key={field.key} className="space-y-1.5">
              <label
                htmlFor={`${name}-${field.key}`}
                className="text-sm font-medium text-foreground"
              >
                {field.label}
              </label>
              <Input
                id={`${name}-${field.key}`}
                type={field.type}
                placeholder={field.placeholder}
                value={fields[field.key]}
                disabled={disabled}
                onChange={(e) => setField(field.key, e.target.value)}
              />
            </div>
          ))}
        </div>

        <label className="flex items-center gap-2 text-sm font-medium text-foreground">
          <input
            type="checkbox"
            className="size-4 rounded border-input accent-primary"
            checked={fields.secure}
            disabled={disabled}
            onChange={(e) => setField("secure", e.target.checked)}
          />
          Use TLS (secure)
        </label>

        <div className="space-y-1.5">
          <label
            htmlFor={`${name}-version`}
            className="text-sm font-medium text-foreground"
          >
            Version
          </label>
          <Input
            id={`${name}-version`}
            placeholder="latest"
            value={fields.version}
            disabled={disabled}
            onChange={(e) => setField("version", e.target.value)}
          />
        </div>

        <div className="space-y-1.5">
          <label
            htmlFor={`${name}-buckets`}
            className="text-sm font-medium text-foreground"
          >
            Buckets
          </label>
          <Input
            id={`${name}-buckets`}
            placeholder="uploads, exports"
            value={fields.buckets}
            disabled={disabled}
            onChange={(e) => setField("buckets", e.target.value)}
          />
          <p className="text-xs text-muted-foreground">
            Comma-separated bucket names to create on apply.
          </p>
        </div>

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
      </CardContent>
    </Card>
  );
}

/** Read the editable field values from the draft rows. */
function readState(rows: ConfigRow[]): MinioFields {
  const map = rowsToMap(rows);
  return {
    host: map.host ?? "",
    port: map.port ?? "",
    consolePort: map.consolePort ?? "",
    user: map.user ?? "",
    password: map.password ?? "",
    version: map.version ?? "",
    buckets: map.buckets ?? "",
    secure: (map.secure ?? "").trim().toLowerCase() === "true",
  };
}

/**
 * Project the fields back onto draft rows, dropping empty values and preserving
 * any unknown rows. `secure` is only written when enabled (its default is
 * false), keeping the saved config free of redundant flags.
 */
function fieldsToRows(fields: MinioFields, prevRows: ConfigRow[]): ConfigRow[] {
  const preserved = prevRows.filter((r) => !KNOWN_KEYS.has(r.key.trim()));
  const known: ConfigRow[] = [];
  const text: (keyof MinioFields)[] = [
    "host",
    "port",
    "consolePort",
    "user",
    "password",
    "version",
    "buckets",
  ];
  for (const key of text) {
    const value = fields[key] as string;
    if (value.trim() !== "") known.push({ key, value });
  }
  if (fields.secure) known.push({ key: "secure", value: "true" });
  return [...known, ...preserved];
}

/** Build the env config object posted to the health check from the draft rows. */
function rowsToConfig(rows: ConfigRow[]): Record<string, unknown> {
  const config: Record<string, unknown> = {};
  for (const { key, value } of rows) {
    const trimmed = key.trim();
    if (trimmed) config[trimmed] = value;
  }
  return config;
}

function rowsToMap(rows: ConfigRow[]): Record<string, string> {
  const map: Record<string, string> = {};
  for (const { key, value } of rows) {
    const trimmed = key.trim();
    if (trimmed) map[trimmed] = value;
  }
  return map;
}
