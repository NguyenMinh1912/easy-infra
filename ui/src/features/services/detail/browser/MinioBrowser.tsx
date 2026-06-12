import {
  AlertCircle,
  ChevronRight,
  ChevronsUpDown,
  File,
  Folder,
  HardDrive,
} from "lucide-react";
import { useEffect, useState } from "react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Skeleton } from "@/components/ui/skeleton";
import { useAsync } from "@/hooks/useAsync";
import { listBuckets } from "@/services/api";

import { BucketContents } from "./BucketContents";

interface MinioBrowserProps {
  /** Profile whose saved connection config the listing runs against. */
  profile: string;
  /** Service name within the profile (the API path segment). */
  service: string;
  /**
   * Bucket configured in the profile's service settings, preselected when it
   * exists in the store. Falls back to the first listed bucket otherwise.
   */
  defaultBucket?: string;
}

/**
 * Object browser for one profile's MinIO: lists the store's buckets and lets
 * the user walk the folders and objects within the selected one. A listing
 * failure (store unreachable) comes back inside the response envelope, so it
 * renders as an expected outcome rather than a transport error.
 */
export function MinioBrowser({
  profile,
  service,
  defaultBucket,
}: MinioBrowserProps) {
  const state = useAsync(
    (signal) => listBuckets(profile, service, signal),
    [profile, service],
  );
  const [bucket, setBucket] = useState<string | null>(null);

  // Pick a bucket once the list loads, so the user lands on content rather than
  // an empty pane: prefer the profile's configured bucket, else the first one.
  // Re-selects if the active bucket disappears.
  const buckets = state.status === "success" ? state.data.buckets : undefined;
  useEffect(() => {
    if (!buckets) return;
    if (buckets.length === 0) {
      setBucket(null);
    } else if (bucket === null || !buckets.includes(bucket)) {
      const preferred =
        defaultBucket && buckets.includes(defaultBucket)
          ? defaultBucket
          : buckets[0];
      setBucket(preferred);
    }
  }, [buckets, bucket, defaultBucket]);

  if (state.status === "loading") {
    return <Skeleton className="h-48 w-full" />;
  }
  if (state.status === "error") {
    return (
      <Alert variant="destructive">
        <AlertCircle />
        <div>
          <AlertTitle>Could not load buckets</AlertTitle>
          <AlertDescription>{state.error.message}</AlertDescription>
        </div>
      </Alert>
    );
  }
  if (state.data.error) {
    return (
      <Alert variant="destructive">
        <AlertCircle />
        <div>
          <AlertTitle>MinIO unreachable</AlertTitle>
          <AlertDescription className="font-mono text-xs">
            {state.data.error}
          </AlertDescription>
        </div>
      </Alert>
    );
  }
  if (state.data.buckets.length === 0) {
    return (
      <div className="flex flex-col items-center gap-3 rounded-md border border-dashed border-border p-10 text-center">
        <span className="flex size-10 items-center justify-center rounded-lg bg-muted">
          <HardDrive className="size-5 text-muted-foreground" aria-hidden />
        </span>
        <p className="text-sm text-muted-foreground">
          This MinIO has no buckets yet.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="w-full justify-between gap-2 sm:w-64"
            aria-label="Select bucket"
          >
            <span className="flex min-w-0 items-center gap-2">
              <Folder
                className="size-4 shrink-0 text-muted-foreground"
                aria-hidden
              />
              <span className="truncate">{bucket}</span>
            </span>
            <ChevronsUpDown
              className="size-4 shrink-0 text-muted-foreground"
              aria-hidden
            />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent
          align="start"
          className="max-h-72 w-[var(--radix-dropdown-menu-trigger-width)] overflow-y-auto"
        >
          {state.data.buckets.map((name) => (
            <DropdownMenuItem
              key={name}
              onSelect={() => setBucket(name)}
              className="gap-2"
            >
              <Folder
                className="size-4 shrink-0 text-muted-foreground"
                aria-hidden
              />
              <span className="truncate">{name}</span>
            </DropdownMenuItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>
      {bucket && (
        // Keyed so switching bucket resets the folder path and reloads.
        <BucketContents
          key={bucket}
          profile={profile}
          service={service}
          bucket={bucket}
        />
      )}
    </div>
  );
}

/** A breadcrumb row for the current path within a bucket. */
export function PathBreadcrumb({
  bucket,
  prefix,
  onNavigate,
}: {
  bucket: string;
  prefix: string;
  onNavigate: (prefix: string) => void;
}) {
  // "a/b/c/" -> segments ["a","b","c"], each linking to its own prefix.
  const segments = prefix.split("/").filter(Boolean);
  let acc = "";
  return (
    <nav
      aria-label="Path"
      className="flex flex-wrap items-center gap-1 text-sm text-muted-foreground"
    >
      <button
        type="button"
        className="font-medium text-foreground hover:underline"
        onClick={() => onNavigate("")}
      >
        {bucket}
      </button>
      {segments.map((segment) => {
        acc += `${segment}/`;
        const target = acc;
        return (
          <span key={target} className="flex items-center gap-1">
            <ChevronRight className="size-3.5" aria-hidden />
            <button
              type="button"
              className="hover:text-foreground hover:underline"
              onClick={() => onNavigate(target)}
            >
              {segment}
            </button>
          </span>
        );
      })}
    </nav>
  );
}

/** Renders a folder or file row icon. */
export function EntryIcon({ folder }: { folder: boolean }) {
  return folder ? (
    <Folder className="size-4 shrink-0 text-muted-foreground" aria-hidden />
  ) : (
    <File className="size-4 shrink-0 text-muted-foreground" aria-hidden />
  );
}
