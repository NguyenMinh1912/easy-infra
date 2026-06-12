import {
  AlertCircle,
  ChevronRight,
  File,
  Folder,
  HardDrive,
} from "lucide-react";
import { useEffect, useState } from "react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useAsync } from "@/hooks/useAsync";
import { listBuckets } from "@/services/api";
import { cn } from "@/lib/utils";

import { BucketContents } from "./BucketContents";

interface MinioBrowserProps {
  /** Profile whose saved connection config the listing runs against. */
  profile: string;
  /** Service name within the profile (the API path segment). */
  service: string;
}

/**
 * Object browser for one profile's MinIO: lists the store's buckets and lets
 * the user walk the folders and objects within the selected one. A listing
 * failure (store unreachable) comes back inside the response envelope, so it
 * renders as an expected outcome rather than a transport error.
 */
export function MinioBrowser({ profile, service }: MinioBrowserProps) {
  const state = useAsync(
    (signal) => listBuckets(profile, service, signal),
    [profile, service],
  );
  const [bucket, setBucket] = useState<string | null>(null);

  // Select the first bucket once the list loads, so the user lands on content
  // rather than an empty pane. Re-selects if the active bucket disappears.
  const buckets = state.status === "success" ? state.data.buckets : undefined;
  useEffect(() => {
    if (!buckets) return;
    if (buckets.length === 0) {
      setBucket(null);
    } else if (bucket === null || !buckets.includes(bucket)) {
      setBucket(buckets[0]);
    }
  }, [buckets, bucket]);

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
    <div className="grid gap-4 sm:grid-cols-[12rem_1fr]">
      <nav aria-label="Buckets" className="flex flex-col gap-1">
        {state.data.buckets.map((name) => (
          <Button
            key={name}
            type="button"
            variant={name === bucket ? "secondary" : "ghost"}
            size="sm"
            className={cn("justify-start gap-2", name !== bucket && "font-normal")}
            onClick={() => setBucket(name)}
          >
            <Folder className="size-4 shrink-0 text-muted-foreground" aria-hidden />
            <span className="truncate">{name}</span>
          </Button>
        ))}
      </nav>
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
