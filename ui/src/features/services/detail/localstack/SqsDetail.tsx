import { useState, type FormEvent } from "react";
import { AlertCircle, Eraser, Inbox, Loader2, Plus, Trash2 } from "lucide-react";
import { toast } from "sonner";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Input } from "@/components/ui/input";
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
import {
  ApiError,
  createQueue,
  deleteQueue,
  listQueues,
  purgeQueue,
} from "@/services/api";
import type { QueueInfo, QueuesResponse } from "@/types/localstack";

import type { AwsServiceDetailProps } from "./types";

/**
 * SQS detail page: create, list, purge and delete the profile's queues, read
 * live from LocalStack. An unreachable endpoint comes back inside the response
 * envelope, so it renders as an expected outcome rather than a transport error.
 * Without a profile (no connection env) it explains how to connect.
 */
export function SqsDetail({ service, profile }: AwsServiceDetailProps) {
  if (!profile) {
    return <NoConnection />;
  }
  return <SqsQueues profile={profile} service={service.id} />;
}

function SqsQueues({ profile, service }: { profile: string; service: string }) {
  // Bumped after a mutation so the listing reloads and reflects the change.
  const [reloadKey, setReloadKey] = useState(0);
  const reload = () => setReloadKey((k) => k + 1);

  const state = useAsync(
    (signal) => listQueues(profile, service, signal),
    [profile, service, reloadKey],
  );

  const queues =
    state.status === "success" && !state.data.error ? state.data.queues : [];

  return (
    <div className="flex flex-col gap-6">
      <CreateQueueForm
        existingNames={queues.map((q) => q.name)}
        onCreate={async (name) => {
          await createQueue(profile, service, name);
          reload();
        }}
      />
      <QueueList
        state={state}
        profile={profile}
        service={service}
        onChanged={reload}
      />
    </div>
  );
}

/** The listing area: loading / error / empty / the queue table. */
function QueueList({
  state,
  profile,
  service,
  onChanged,
}: {
  state: ReturnType<typeof useAsync<QueuesResponse>>;
  profile: string;
  service: string;
  onChanged: () => void;
}) {
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
              No queues yet. Create one with the form above and it will appear
              here.
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
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {queues.map((q) => (
              <QueueRow
                key={q.url}
                queue={q}
                profile={profile}
                service={service}
                onChanged={onChanged}
              />
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}

/** One queue row, with purge and delete actions. */
function QueueRow({
  queue,
  profile,
  service,
  onChanged,
}: {
  queue: QueueInfo;
  profile: string;
  service: string;
  onChanged: () => void;
}) {
  const [busy, setBusy] = useState(false);

  async function run(
    action: () => Promise<void>,
    success: string,
    failure: string,
  ) {
    setBusy(true);
    try {
      await action();
      toast.success(success);
      onChanged();
    } catch (cause) {
      toast.error(failure, {
        description: cause instanceof ApiError ? cause.message : String(cause),
      });
    } finally {
      setBusy(false);
    }
  }

  return (
    <TableRow>
      <TableCell className="font-mono">{queue.name}</TableCell>
      <TableCell className="text-right tabular-nums">{queue.messages}</TableCell>
      <TableCell className="text-right tabular-nums">{queue.inFlight}</TableCell>
      <TableCell className="text-right">
        <div className="flex justify-end gap-2">
          <ConfirmDialog
            title={`Purge "${queue.name}"?`}
            description="This removes all messages from the queue. The queue itself is kept. This cannot be undone."
            confirmLabel="Purge"
            variant="destructive"
            onConfirm={() =>
              void run(
                () => purgeQueue(profile, service, queue.url),
                `Purged "${queue.name}"`,
                "Could not purge queue",
              )
            }
            trigger={
              <Button variant="ghost" size="sm" disabled={busy}>
                {busy ? (
                  <Loader2 className="animate-spin" aria-hidden />
                ) : (
                  <Eraser aria-hidden />
                )}
                Purge
              </Button>
            }
          />
          <ConfirmDialog
            title={`Delete "${queue.name}"?`}
            description="This permanently deletes the queue and all of its messages. This cannot be undone."
            confirmLabel="Delete"
            variant="destructive"
            onConfirm={() =>
              void run(
                () => deleteQueue(profile, service, queue.url),
                `Deleted "${queue.name}"`,
                "Could not delete queue",
              )
            }
            trigger={
              <Button variant="ghost" size="sm" disabled={busy}>
                {busy ? (
                  <Loader2 className="animate-spin" aria-hidden />
                ) : (
                  <Trash2 aria-hidden />
                )}
                Delete
              </Button>
            }
          />
        </div>
      </TableCell>
    </TableRow>
  );
}

interface CreateQueueFormProps {
  /** Names of queues that already exist, for duplicate validation. */
  existingNames: string[];
  onCreate: (name: string) => Promise<void>;
}

/**
 * Form for creating a queue. Validates the name client-side (non-empty, not a
 * duplicate) and surfaces the result as a toast. A name ending in `.fifo`
 * creates a FIFO queue, which LocalStack enforces server-side.
 */
function CreateQueueForm({ existingNames, onCreate }: CreateQueueFormProps) {
  const [name, setName] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const trimmed = name.trim();
  const duplicate = existingNames.some(
    (existing) => existing.toLowerCase() === trimmed.toLowerCase(),
  );
  const validationError =
    trimmed.length > 0 && duplicate
      ? `A queue named "${trimmed}" already exists.`
      : null;
  const canSubmit = trimmed.length > 0 && !duplicate && !submitting;

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    if (!canSubmit) return;
    setSubmitting(true);
    try {
      await onCreate(trimmed);
      toast.success(`Queue "${trimmed}" created`);
      setName("");
    } catch (cause) {
      toast.error("Could not create queue", {
        description: cause instanceof ApiError ? cause.message : String(cause),
      });
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">New queue</CardTitle>
        <CardDescription>
          Creates an SQS queue. End the name with <code>.fifo</code> for a FIFO
          queue.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form
          onSubmit={handleSubmit}
          className="flex flex-col gap-3 sm:flex-row"
        >
          <label htmlFor="queue-name" className="sr-only">
            Queue name
          </label>
          <Input
            id="queue-name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g. orders"
            disabled={submitting}
            aria-invalid={validationError !== null}
            aria-describedby={validationError ? "queue-name-error" : undefined}
            className="sm:max-w-xs"
          />
          <Button type="submit" disabled={!canSubmit}>
            {submitting ? (
              <Loader2 className="animate-spin" aria-hidden />
            ) : (
              <Plus aria-hidden />
            )}
            {submitting ? "Creating…" : "Create queue"}
          </Button>
        </form>
        {validationError && (
          <p
            id="queue-name-error"
            role="alert"
            className="mt-3 text-sm text-destructive"
          >
            {validationError}
          </p>
        )}
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
