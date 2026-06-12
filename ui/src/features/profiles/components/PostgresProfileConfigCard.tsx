import { useMemo, useState } from "react";
import { CheckCircle2, Loader2, PlugZap, XCircle } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { checkServiceConnection } from "@/services/api";

import {
  buildPostgresUrl,
  parsePostgresUrl,
  type PostgresFields,
} from "../postgres-url";
import type { ConfigRow } from "./ProfileServiceConfigCard";
import type { ProfileServiceConfigCardProps } from "./ProfileServiceConfigCard";

/** Discrete fields this card edits, in display order. */
const FIELD_KEYS = [
  "host",
  "port",
  "user",
  "password",
  "database",
  "schema",
] as const satisfies readonly (keyof PostgresFields)[];

/** Keys this card owns; rows with any other key are preserved untouched. */
const KNOWN_KEYS = new Set<string>([...FIELD_KEYS, "url"]);

/** Per-field labels and input hints. */
const FIELD_META: Record<
  keyof PostgresFields,
  { label: string; placeholder: string; type?: string }
> = {
  host: { label: "Host", placeholder: "localhost" },
  port: { label: "Port", placeholder: "5432" },
  user: { label: "User", placeholder: "app" },
  password: { label: "Password", placeholder: "", type: "password" },
  database: { label: "Database", placeholder: "app" },
  schema: { label: "Schema", placeholder: "public" },
};

type CheckState =
  | { status: "idle" }
  | { status: "checking" }
  | { status: "ok" }
  | { status: "error"; message: string };

/**
 * Tailored profile config editor for postgres. Surfaces the connection as a
 * single connection string *and* discrete fields with two-way binding: pasting
 * a connection string fills the fields, and editing a field updates the
 * connection string. The schema is a first-class field. A "Check connection"
 * button probes the database with the current (unsaved) values.
 *
 * Like the generic card it owns no persisted state — the parent holds the rows
 * draft — so it canonicalises to discrete fields on every change.
 */
export function PostgresProfileConfigCard({
  name,
  profileName,
  rows,
  onChange,
  disabled,
}: ProfileServiceConfigCardProps) {
  const { fields, loadedExtras } = useMemo(() => readState(rows), [rows]);
  // While the user is typing a custom connection string we keep the raw text
  // here; null means "mirror the discrete fields".
  const [urlDraft, setUrlDraft] = useState<string | null>(null);
  const [check, setCheck] = useState<CheckState>({ status: "idle" });

  const connectionString = urlDraft ?? buildPostgresUrl(fields);
  const draftParse =
    urlDraft && urlDraft.trim() !== "" ? parsePostgresUrl(urlDraft) : null;
  const urlInvalid =
    urlDraft != null && urlDraft.trim() !== "" && draftParse === null;
  const extraParams = draftParse?.extraParams ?? loadedExtras;

  const setField = (key: keyof PostgresFields, value: string) => {
    setCheck({ status: "idle" });
    setUrlDraft(null);
    onChange(fieldsToRows({ ...fields, [key]: value }, rows));
  };

  const setConnectionString = (value: string) => {
    setCheck({ status: "idle" });
    // Empty reverts to mirroring the fields rather than blanking them.
    setUrlDraft(value === "" ? null : value);
    const parsed = parsePostgresUrl(value);
    if (parsed) {
      const { extraParams: _ignored, ...parsedFields } = parsed;
      onChange(fieldsToRows(parsedFields, rows));
    }
  };

  const runCheck = async () => {
    setCheck({ status: "checking" });
    try {
      const result = await checkServiceConnection(
        profileName,
        name,
        fieldsToConfig(fields),
      );
      setCheck(
        result.ok
          ? { status: "ok" }
          : { status: "error", message: result.error ?? "connection failed" },
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
        <div className="space-y-1.5">
          <label
            htmlFor={`${name}-url`}
            className="text-sm font-medium text-foreground"
          >
            Connection string
          </label>
          <Input
            id={`${name}-url`}
            className="font-mono text-sm"
            placeholder="postgresql://app:secret@localhost:5432/app?search_path=public"
            value={connectionString}
            disabled={disabled}
            aria-invalid={urlInvalid}
            onChange={(e) => setConnectionString(e.target.value)}
          />
          <p className="text-xs text-muted-foreground">
            Paste a connection string to fill the fields below, or edit the
            fields to build one. <code className="font-mono">jdbc:</code> URLs
            and <code className="font-mono">currentSchema</code> are accepted.
          </p>
          {urlInvalid && (
            <p className="text-xs text-destructive">
              Not a valid postgres connection string yet.
            </p>
          )}
          {extraParams.length > 0 && (
            <p className="text-xs text-amber-600 dark:text-amber-500">
              Extra parameters not stored as fields:{" "}
              <span className="font-mono">{extraParams.join(", ")}</span>.
            </p>
          )}
        </div>

        <div className="grid gap-3 sm:grid-cols-2">
          {FIELD_KEYS.map((key) => {
            const meta = FIELD_META[key];
            return (
              <div key={key} className="space-y-1.5">
                <label
                  htmlFor={`${name}-${key}`}
                  className="text-sm font-medium text-foreground"
                >
                  {meta.label}
                </label>
                <Input
                  id={`${name}-${key}`}
                  type={meta.type}
                  placeholder={meta.placeholder}
                  value={fields[key]}
                  disabled={disabled}
                  onChange={(e) => setField(key, e.target.value)}
                />
              </div>
            );
          })}
        </div>

        <div className="flex items-center gap-3">
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={busy || urlInvalid || fields.host.trim() === ""}
            onClick={runCheck}
          >
            {check.status === "checking" ? (
              <Loader2 className="animate-spin" aria-hidden />
            ) : (
              <PlugZap aria-hidden />
            )}
            Check connection
          </Button>
          {check.status === "ok" && (
            <span className="flex items-center gap-1.5 text-sm text-emerald-600 dark:text-emerald-500">
              <CheckCircle2 className="size-4" aria-hidden />
              Connected
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
function readState(rows: ConfigRow[]): {
  fields: PostgresFields;
  loadedExtras: string[];
} {
  const map = rowsToMap(rows);
  const discrete: PostgresFields = {
    host: map.host ?? "",
    port: map.port ?? "",
    user: map.user ?? "",
    password: map.password ?? "",
    database: map.database ?? "",
    schema: map.schema ?? "",
  };

  // A profile may store the connection as a single `url` instead of discrete
  // fields; when that's all there is, extract it so the form is populated.
  const hasDiscrete = FIELD_KEYS.some((k) => (map[k] ?? "").trim() !== "");
  if (!hasDiscrete && map.url) {
    const parsed = parsePostgresUrl(map.url);
    if (parsed) {
      const { extraParams, ...fields } = parsed;
      return { fields, loadedExtras: extraParams };
    }
  }
  return { fields: discrete, loadedExtras: [] };
}

/**
 * Project the fields back onto draft rows, dropping empties and preserving any
 * unknown rows. Canonicalises to discrete fields (no `url`), so the saved
 * config has one representation regardless of how it was entered.
 */
function fieldsToRows(fields: PostgresFields, prevRows: ConfigRow[]): ConfigRow[] {
  const preserved = prevRows.filter((r) => !KNOWN_KEYS.has(r.key.trim()));
  const rows = FIELD_KEYS.filter((k) => fields[k].trim() !== "").map((k) => ({
    key: k,
    value: fields[k],
  }));
  return [...rows, ...preserved];
}

/** The env config object posted to the connection check. */
function fieldsToConfig(fields: PostgresFields): Record<string, unknown> {
  const config: Record<string, unknown> = {};
  for (const key of FIELD_KEYS) {
    if (fields[key].trim() !== "") config[key] = fields[key];
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
