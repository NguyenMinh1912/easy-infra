import { useEffect, useState } from "react";
import { AlertCircle, Play } from "lucide-react";
import { toast } from "sonner";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import {
  ApiError,
  getProfileConfig,
  getTemplate,
  listProfiles,
  runTemplate,
} from "@/services/api";
import type { QueryResult } from "@/types/console";

import { QueryResultTable } from "@/features/services/detail/console/QueryResultTable";
import { renderSql } from "../variables";

interface TemplateRunDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Name of the template to run, or null when the dialog is closed. */
  templateName: string | null;
}

interface ServiceOption {
  id: string;
  name: string;
}

const selectClass = cn(
  "h-9 w-full rounded-md border border-input bg-background px-3 text-sm shadow-sm",
  "transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
);

/**
 * Fill in a template's variables, pick a target profile/service, preview the
 * rendered SQL, and run it. Results reuse the console's result table (so a
 * single-table result is even editable in place). The server performs the
 * authoritative render and execution.
 */
export function TemplateRunDialog({
  open,
  onOpenChange,
  templateName,
}: TemplateRunDialogProps) {
  const [sql, setSql] = useState("");
  const [variables, setVariables] = useState<string[]>([]);
  const [values, setValues] = useState<Record<string, string>>({});

  const [profiles, setProfiles] = useState<string[]>([]);
  const [profile, setProfile] = useState("");
  const [services, setServices] = useState<ServiceOption[]>([]);
  const [service, setService] = useState("");

  const [result, setResult] = useState<QueryResult | null>(null);
  const [busy, setBusy] = useState(false);

  // Load the template body and the profile list when the dialog opens.
  useEffect(() => {
    if (!open || templateName === null) return;
    setResult(null);
    const controller = new AbortController();
    Promise.all([
      getTemplate(templateName, controller.signal),
      listProfiles(controller.signal),
    ])
      .then(([t, p]) => {
        setSql(t.sql);
        setVariables(t.variables);
        setValues(Object.fromEntries(t.variables.map((v) => [v, ""])));
        setProfiles(p.profiles.map((pr) => pr.name));
        setProfile(p.activeProfile || p.profiles[0]?.name || "");
      })
      .catch((cause: unknown) => {
        if (controller.signal.aborted) return;
        toast.error("Could not load template", {
          description: cause instanceof ApiError ? cause.message : String(cause),
        });
      });
    return () => controller.abort();
  }, [open, templateName]);

  // Load the selected profile's services whenever the profile changes.
  useEffect(() => {
    if (!open || profile === "") return;
    const controller = new AbortController();
    getProfileConfig(profile, controller.signal)
      .then((cfg) => {
        const opts = cfg.services.map((s) => ({ id: s.id, name: s.name }));
        setServices(opts);
        setService(opts[0]?.id ?? "");
      })
      .catch(() => {
        if (controller.signal.aborted) return;
        setServices([]);
        setService("");
      });
    return () => controller.abort();
  }, [open, profile]);

  const preview = renderSql(sql, values);

  const run = async () => {
    if (templateName === null) return;
    setBusy(true);
    try {
      const res = await runTemplate(templateName, { profile, service, variables: values });
      setResult(res);
    } catch (cause) {
      toast.error("Run failed", {
        description: cause instanceof ApiError ? cause.message : String(cause),
      });
    } finally {
      setBusy(false);
    }
  };

  const canRun = profile !== "" && service !== "" && !busy;

  return (
    <Dialog open={open} onOpenChange={(o) => !busy && onOpenChange(o)}>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle>Run: {templateName}</DialogTitle>
          <DialogDescription>
            Fill in the variables and choose where to run. Values are substituted
            into the SQL as written.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <label htmlFor="run-profile" className="text-sm font-medium">
                Profile
              </label>
              <select
                id="run-profile"
                className={selectClass}
                value={profile}
                onChange={(e) => setProfile(e.target.value)}
              >
                {profiles.map((p) => (
                  <option key={p} value={p}>
                    {p}
                  </option>
                ))}
              </select>
            </div>
            <div className="space-y-1.5">
              <label htmlFor="run-service" className="text-sm font-medium">
                Service
              </label>
              <select
                id="run-service"
                className={selectClass}
                value={service}
                onChange={(e) => setService(e.target.value)}
              >
                {services.length === 0 && <option value="">No services</option>}
                {services.map((s) => (
                  <option key={s.id} value={s.id}>
                    {s.name}
                  </option>
                ))}
              </select>
            </div>
          </div>

          {variables.length > 0 && (
            <div className="space-y-2">
              <p className="text-sm font-medium">Variables</p>
              {variables.map((v) => (
                <div key={v} className="grid grid-cols-[8rem_1fr] items-center gap-2">
                  <label
                    htmlFor={`var-${v}`}
                    className="truncate font-mono text-sm text-muted-foreground"
                  >
                    {v}
                  </label>
                  <Input
                    id={`var-${v}`}
                    value={values[v] ?? ""}
                    onChange={(e) =>
                      setValues((prev) => ({ ...prev, [v]: e.target.value }))
                    }
                  />
                </div>
              ))}
            </div>
          )}

          <div className="space-y-1.5">
            <p className="text-sm font-medium">Rendered SQL</p>
            <pre className="max-h-40 overflow-auto whitespace-pre-wrap break-words rounded-md border border-border bg-muted/30 p-3 font-mono text-xs">
              {preview || "—"}
            </pre>
          </div>

          <div className="flex justify-end">
            <Button disabled={!canRun} onClick={() => void run()}>
              <Play aria-hidden />
              Run
            </Button>
          </div>

          {result &&
            (result.error ? (
              <Alert variant="destructive">
                <AlertCircle />
                <div>
                  <AlertTitle>Statement failed</AlertTitle>
                  <AlertDescription>{result.error}</AlertDescription>
                </div>
              </Alert>
            ) : (
              <QueryResultTable
                result={result}
                profile={profile}
                service={service}
                onChanged={() => void run()}
              />
            ))}
        </div>
      </DialogContent>
    </Dialog>
  );
}
