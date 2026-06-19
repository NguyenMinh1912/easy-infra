import { useState, type FormEvent } from "react";
import { AlertCircle, AtSign, Loader2, Plus, Trash2 } from "lucide-react";
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
  createIdentity,
  deleteIdentity,
  listIdentities,
} from "@/services/api";
import type { IdentitiesResponse, IdentityInfo } from "@/types/localstack";

import type { AwsServiceDetailProps } from "./types";

/**
 * SES detail page: create, list and delete the profile's verified email and
 * domain identities, read live from LocalStack. An unreachable endpoint comes
 * back inside the response envelope, so it renders as an expected outcome rather
 * than a transport error. Without a profile (no connection env) it explains how
 * to connect.
 */
export function SesDetail({ service, profile, region }: AwsServiceDetailProps) {
  if (!profile) {
    return <NoConnection />;
  }
  return (
    <SesIdentities profile={profile} service={service.id} region={region} />
  );
}

function SesIdentities({
  profile,
  service,
  region,
}: {
  profile: string;
  service: string;
  region?: string;
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
}: {
  state: ReturnType<typeof useAsync<IdentitiesResponse>>;
  profile: string;
  service: string;
  region?: string;
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
              />
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}

/** One identity row, with a delete action. */
function IdentityRow({
  identity,
  profile,
  service,
  region,
  onChanged,
}: {
  identity: IdentityInfo;
  profile: string;
  service: string;
  region?: string;
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
      <TableCell className="font-mono">{identity.identity}</TableCell>
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
