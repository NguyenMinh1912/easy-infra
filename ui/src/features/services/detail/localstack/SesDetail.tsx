import { useState, type FormEvent } from "react";
import {
  AlertCircle,
  ArrowLeft,
  AtSign,
  Loader2,
  Mail,
  Plus,
  RotateCw,
  Trash2,
} from "lucide-react";
import { toast } from "sonner";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
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
import { useHashRoute } from "@/hooks/useHashRoute";
import {
  ApiError,
  createIdentity,
  deleteIdentity,
  listIdentities,
  listMessages,
} from "@/services/api";
import type {
  IdentitiesResponse,
  IdentityInfo,
  MessageInfo,
  MessagesResponse,
} from "@/types/localstack";

import type { AwsServiceDetailProps } from "./types";

/**
 * SES detail page: create, list and delete the profile's verified email and
 * domain identities, read live from LocalStack. Opening an identity drills into
 * its mail list (a deep-linkable `…/ses/{identity}` sub-route). An unreachable
 * endpoint comes back inside the response envelope, so it renders as an expected
 * outcome rather than a transport error. Without a profile (no connection env)
 * it explains how to connect.
 */
export function SesDetail({ service, profile, region }: AwsServiceDetailProps) {
  const route = useHashRoute();
  if (!profile) {
    return <NoConnection />;
  }

  // Sub-route for a single identity's mail list, deep-linkable so the back
  // button works: #/profiles/{p}/services/{s}/ses/{identity}.
  const sesPath = `/profiles/${encodeURIComponent(profile)}/services/${encodeURIComponent(service.id)}/ses`;
  const selectedIdentity = route.startsWith(`${sesPath}/`)
    ? decodeURIComponent(route.slice(sesPath.length + 1))
    : null;
  const navigate = (identity: string | null) => {
    window.location.hash = identity
      ? `${sesPath}/${encodeURIComponent(identity)}`
      : sesPath;
  };

  if (selectedIdentity) {
    return (
      <IdentityMail
        identity={selectedIdentity}
        profile={profile}
        service={service.id}
        region={region}
        onBack={() => navigate(null)}
      />
    );
  }

  return (
    <SesIdentities
      profile={profile}
      service={service.id}
      region={region}
      onOpenIdentity={(identity) => navigate(identity)}
    />
  );
}

function SesIdentities({
  profile,
  service,
  region,
  onOpenIdentity,
}: {
  profile: string;
  service: string;
  region?: string;
  onOpenIdentity: (identity: string) => void;
}) {
  // Bumped after a mutation so the listing reloads and reflects the change.
  const [reloadKey, setReloadKey] = useState(0);
  const reload = () => setReloadKey((k) => k + 1);

  const state = useAsync(
    (signal) => listIdentities(profile, service, region, signal),
    [profile, service, region, reloadKey],
  );

  const identities =
    state.status === "success" && !state.data.error
      ? state.data.identities
      : [];

  return (
    <div className="flex flex-col gap-6">
      <CreateIdentityForm
        existingNames={identities.map((id) => id.identity)}
        onCreate={async (identity) => {
          await createIdentity(profile, service, identity, region);
          reload();
        }}
      />
      <IdentityList
        state={state}
        profile={profile}
        service={service}
        region={region}
        onChanged={reload}
        onOpenIdentity={onOpenIdentity}
      />
    </div>
  );
}

/** The listing area: loading / error / empty / the identity table. */
function IdentityList({
  state,
  profile,
  service,
  region,
  onChanged,
  onOpenIdentity,
}: {
  state: ReturnType<typeof useAsync<IdentitiesResponse>>;
  profile: string;
  service: string;
  region?: string;
  onChanged: () => void;
  onOpenIdentity: (identity: string) => void;
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

  const identities = state.data.identities;
  if (identities.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Identities</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col items-center gap-3 py-8 text-center">
            <span className="flex size-10 items-center justify-center rounded-lg bg-muted">
              <AtSign className="size-5 text-muted-foreground" aria-hidden />
            </span>
            <p className="max-w-md text-sm text-muted-foreground">
              No identities yet. Verify an email or domain with the form above
              and it will appear here.
            </p>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">
          Identities ({identities.length})
        </CardTitle>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Identity</TableHead>
              <TableHead>Type</TableHead>
              <TableHead>Status</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {identities.map((id) => (
              <IdentityRow
                key={id.identity}
                identity={id}
                profile={profile}
                service={service}
                region={region}
                onChanged={onChanged}
                onOpen={() => onOpenIdentity(id.identity)}
              />
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}

/** One identity row, opening its mail list, with a delete action. */
function IdentityRow({
  identity,
  profile,
  service,
  region,
  onChanged,
  onOpen,
}: {
  identity: IdentityInfo;
  profile: string;
  service: string;
  region?: string;
  onChanged: () => void;
  onOpen: () => void;
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
      <TableCell className="font-mono">
        <button
          type="button"
          onClick={onOpen}
          className="text-left font-medium text-foreground underline-offset-4 hover:underline focus-visible:underline focus-visible:outline-none"
          aria-label={`View mail for ${identity.identity}`}
        >
          {identity.identity}
        </button>
      </TableCell>
      <TableCell>
        <Badge variant="secondary">{identityTypeLabel(identity.type)}</Badge>
      </TableCell>
      <TableCell>
        <Badge variant={identity.verified ? "success" : "outline"}>
          {identity.verified ? "Verified" : "Pending"}
        </Badge>
      </TableCell>
      <TableCell className="text-right">
        <div className="flex justify-end gap-2">
          <Button variant="ghost" size="sm" onClick={onOpen} disabled={busy}>
            <Mail aria-hidden />
            View mail
          </Button>
          <ConfirmDialog
            title={`Delete "${identity.identity}"?`}
            description="This removes the identity from SES. It will no longer be able to send email. This cannot be undone."
            confirmLabel="Delete"
            variant="destructive"
            onConfirm={() =>
              void run(
                () =>
                  deleteIdentity(profile, service, identity.identity, region),
                `Deleted "${identity.identity}"`,
                "Could not delete identity",
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

/**
 * An identity's mail list: the SES messages LocalStack recorded that involve the
 * identity (as sender or recipient), newest first. Reuses the listing's
 * loading / unreachable / empty states; a row opens the full message. An
 * unreachable endpoint comes back inside the response envelope, like the
 * identity listing.
 */
function IdentityMail({
  identity,
  profile,
  service,
  region,
  onBack,
}: {
  identity: string;
  profile: string;
  service: string;
  region?: string;
  onBack: () => void;
}) {
  // Bumped to reload after the user retries an unreachable endpoint.
  const [reloadKey, setReloadKey] = useState(0);
  const [selected, setSelected] = useState<MessageInfo | null>(null);

  const state = useAsync(
    (signal) => listMessages(profile, service, identity, region, signal),
    [profile, service, identity, region, reloadKey],
  );

  return (
    <div className="flex flex-col gap-6">
      <Button
        variant="ghost"
        size="sm"
        className="-ml-2 w-fit gap-1.5 text-muted-foreground"
        onClick={onBack}
      >
        <ArrowLeft className="size-4" aria-hidden />
        Identities
      </Button>

      <MailList state={state} identity={identity} onOpen={setSelected} />

      <MessageDialog
        message={selected}
        onClose={() => setSelected(null)}
      />

      {state.status === "error" && (
        <Button
          variant="outline"
          size="sm"
          className="w-fit"
          onClick={() => setReloadKey((k) => k + 1)}
        >
          <RotateCw aria-hidden />
          Retry
        </Button>
      )}
    </div>
  );
}

/** The mail area: loading / error / empty / the message table. */
function MailList({
  state,
  identity,
  onOpen,
}: {
  state: ReturnType<typeof useAsync<MessagesResponse>>;
  identity: string;
  onOpen: (message: MessageInfo) => void;
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

  const messages = state.data.messages;
  if (messages.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base">
            Mail for <span className="font-mono">{identity}</span>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col items-center gap-3 py-8 text-center">
            <span className="flex size-10 items-center justify-center rounded-lg bg-muted">
              <Mail className="size-5 text-muted-foreground" aria-hidden />
            </span>
            <p className="max-w-md text-sm text-muted-foreground">
              No mail yet. Messages sent to or from this identity through SES
              will appear here.
            </p>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">
          Mail for <span className="font-mono">{identity}</span> (
          {messages.length})
        </CardTitle>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>From</TableHead>
              <TableHead>To</TableHead>
              <TableHead>Subject</TableHead>
              <TableHead className="text-right">Sent</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {messages.map((m) => (
              <MessageRow key={m.id} message={m} onOpen={() => onOpen(m)} />
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}

/** One message row; clicking it opens the full message. */
function MessageRow({
  message,
  onOpen,
}: {
  message: MessageInfo;
  onOpen: () => void;
}) {
  return (
    <TableRow
      className="cursor-pointer"
      onClick={onOpen}
      tabIndex={0}
      role="button"
      aria-label={`Open message "${message.subject || "(no subject)"}"`}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          onOpen();
        }
      }}
    >
      <TableCell className="font-mono">{message.source || "—"}</TableCell>
      <TableCell className="font-mono">
        {message.destination.length > 0 ? message.destination.join(", ") : "—"}
      </TableCell>
      <TableCell>{message.subject || <span className="text-muted-foreground">(no subject)</span>}</TableCell>
      <TableCell className="text-right tabular-nums text-muted-foreground">
        {formatTimestamp(message.timestamp)}
      </TableCell>
    </TableRow>
  );
}

/** A dialog showing one message's headers and body. */
function MessageDialog({
  message,
  onClose,
}: {
  message: MessageInfo | null;
  onClose: () => void;
}) {
  return (
    <Dialog open={message !== null} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-2xl">
        {message && (
          <>
            <DialogHeader>
              <DialogTitle>{message.subject || "(no subject)"}</DialogTitle>
              <DialogDescription asChild>
                <div className="flex flex-col gap-0.5 text-left">
                  <span>
                    <span className="font-medium text-foreground">From:</span>{" "}
                    <span className="font-mono">{message.source || "—"}</span>
                  </span>
                  <span>
                    <span className="font-medium text-foreground">To:</span>{" "}
                    <span className="font-mono">
                      {message.destination.length > 0
                        ? message.destination.join(", ")
                        : "—"}
                    </span>
                  </span>
                  {message.timestamp && (
                    <span>
                      <span className="font-medium text-foreground">Sent:</span>{" "}
                      {formatTimestamp(message.timestamp)}
                    </span>
                  )}
                </div>
              </DialogDescription>
            </DialogHeader>
            <div className="max-h-80 overflow-auto rounded-lg border bg-muted/40 p-3">
              {message.body ? (
                <pre className="whitespace-pre-wrap break-words font-mono text-sm">
                  {message.body}
                </pre>
              ) : (
                <p className="text-sm text-muted-foreground">
                  This message has no text body.
                </p>
              )}
            </div>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}

/** Format an RFC 3339 timestamp for display, leaving unparseable values as-is. */
function formatTimestamp(timestamp: string): string {
  if (!timestamp) return "—";
  const date = new Date(timestamp);
  if (Number.isNaN(date.getTime())) return timestamp;
  return date.toLocaleString();
}

interface CreateIdentityFormProps {
  /** Identities that already exist, for duplicate validation. */
  existingNames: string[];
  onCreate: (identity: string) => Promise<void>;
}

/**
 * Form for verifying a new SES identity. Validates client-side (non-empty, not a
 * duplicate) and surfaces the result as a toast. An identity containing `@` is
 * verified as an email address, otherwise as a domain — LocalStack marks it
 * verified immediately rather than sending a real confirmation.
 */
function CreateIdentityForm({
  existingNames,
  onCreate,
}: CreateIdentityFormProps) {
  const [identity, setIdentity] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const trimmed = identity.trim();
  const duplicate = existingNames.some(
    (existing) => existing.toLowerCase() === trimmed.toLowerCase(),
  );
  const validationError =
    trimmed.length > 0 && duplicate
      ? `An identity "${trimmed}" already exists.`
      : null;
  const canSubmit = trimmed.length > 0 && !duplicate && !submitting;

  async function handleSubmit(event: FormEvent) {
    event.preventDefault();
    if (!canSubmit) return;
    setSubmitting(true);
    try {
      await onCreate(trimmed);
      toast.success(`Identity "${trimmed}" added`);
      setIdentity("");
    } catch (cause) {
      toast.error("Could not add identity", {
        description: cause instanceof ApiError ? cause.message : String(cause),
      });
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">New identity</CardTitle>
        <CardDescription>
          Verifies an SES identity. Enter an email address (e.g.{" "}
          <code>dev@example.com</code>) or a domain (e.g.{" "}
          <code>example.com</code>).
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form
          onSubmit={handleSubmit}
          className="flex flex-col gap-3 sm:flex-row"
        >
          <label htmlFor="identity-name" className="sr-only">
            Email or domain
          </label>
          <Input
            id="identity-name"
            value={identity}
            onChange={(e) => setIdentity(e.target.value)}
            placeholder="e.g. dev@example.com"
            disabled={submitting}
            aria-invalid={validationError !== null}
            aria-describedby={
              validationError ? "identity-name-error" : undefined
            }
            className="sm:max-w-xs"
          />
          <Button type="submit" disabled={!canSubmit}>
            {submitting ? (
              <Loader2 className="animate-spin" aria-hidden />
            ) : (
              <Plus aria-hidden />
            )}
            {submitting ? "Adding…" : "Add identity"}
          </Button>
        </form>
        {validationError && (
          <p
            id="identity-name-error"
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

/** Friendly label for an SES identity type (e.g. EMAIL_ADDRESS → Email). */
function identityTypeLabel(type: string): string {
  switch (type) {
    case "EMAIL_ADDRESS":
      return "Email";
    case "DOMAIN":
    case "MANAGED_DOMAIN":
      return "Domain";
    default:
      return type;
  }
}

/** Shown when the page is not profile-scoped, so there is no endpoint to query. */
function NoConnection() {
  return (
    <Card>
      <CardContent className="py-8 text-center text-sm text-muted-foreground">
        Open this service from a profile to browse its SES identities.
      </CardContent>
    </Card>
  );
}
