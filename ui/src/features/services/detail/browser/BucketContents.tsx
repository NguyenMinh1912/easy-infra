import { AlertCircle } from "lucide-react";
import { useState } from "react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
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
import { listObjects } from "@/services/api";
import type { ObjectEntry } from "@/types/browse";

import { EntryIcon, PathBreadcrumb } from "./MinioBrowser";

interface BucketContentsProps {
  profile: string;
  service: string;
  bucket: string;
}

/**
 * The contents of one bucket at the current folder path: sub-folders (which
 * navigate deeper on click) above the objects at this level, with a breadcrumb
 * back to any ancestor. Owns the in-bucket `prefix` state and reloads whenever
 * it changes.
 */
export function BucketContents({
  profile,
  service,
  bucket,
}: BucketContentsProps) {
  const [prefix, setPrefix] = useState("");
  const state = useAsync(
    (signal) => listObjects(profile, service, bucket, prefix, signal),
    [profile, service, bucket, prefix],
  );

  return (
    <div className="space-y-3">
      <PathBreadcrumb bucket={bucket} prefix={prefix} onNavigate={setPrefix} />
      {state.status === "loading" && (
        <div className="space-y-2" aria-label="Loading objects">
          <Skeleton className="h-8 w-full" />
          <Skeleton className="h-8 w-full" />
          <Skeleton className="h-8 w-2/3" />
        </div>
      )}
      {state.status === "error" && (
        <Alert variant="destructive">
          <AlertCircle />
          <div>
            <AlertTitle>Could not load objects</AlertTitle>
            <AlertDescription>{state.error.message}</AlertDescription>
          </div>
        </Alert>
      )}
      {state.status === "success" &&
        (state.data.error ? (
          <Alert variant="destructive">
            <AlertCircle />
            <div>
              <AlertTitle>MinIO unreachable</AlertTitle>
              <AlertDescription className="font-mono text-xs">
                {state.data.error}
              </AlertDescription>
            </div>
          </Alert>
        ) : (
          <ObjectsTable
            prefixes={state.data.prefixes}
            objects={state.data.objects}
            onOpen={setPrefix}
          />
        ))}
    </div>
  );
}

interface ObjectsTableProps {
  prefixes: string[];
  objects: ObjectEntry[];
  onOpen: (prefix: string) => void;
}

/** The folders and objects of one level, folders first. */
function ObjectsTable({ prefixes, objects, onOpen }: ObjectsTableProps) {
  if (prefixes.length === 0 && objects.length === 0) {
    return (
      <div className="rounded-md border border-dashed border-border p-8 text-center text-sm text-muted-foreground">
        This folder is empty.
      </div>
    );
  }
  return (
    <div className="overflow-x-auto rounded-md border border-border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Name</TableHead>
            <TableHead className="w-28 text-right">Size</TableHead>
            <TableHead className="w-44">Modified</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {prefixes.map((p) => (
            <TableRow
              key={p}
              className="cursor-pointer"
              onClick={() => onOpen(p)}
            >
              <TableCell>
                <button
                  type="button"
                  className="flex items-center gap-2 font-medium hover:underline"
                  onClick={() => onOpen(p)}
                >
                  <EntryIcon folder />
                  {baseName(p)}/
                </button>
              </TableCell>
              <TableCell className="text-right text-muted-foreground">—</TableCell>
              <TableCell className="text-muted-foreground">—</TableCell>
            </TableRow>
          ))}
          {objects.map((obj) => (
            <TableRow key={obj.key}>
              <TableCell>
                <span className="flex items-center gap-2">
                  <EntryIcon folder={false} />
                  <span className="truncate font-mono text-sm" title={obj.key}>
                    {baseName(obj.key)}
                  </span>
                </span>
              </TableCell>
              <TableCell className="text-right font-mono text-sm">
                {formatBytes(obj.size)}
              </TableCell>
              <TableCell className="text-sm text-muted-foreground">
                {formatTime(obj.lastModified)}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

/** The last path segment of a key or prefix ("a/b/c.txt" -> "c.txt"). */
function baseName(key: string): string {
  const trimmed = key.endsWith("/") ? key.slice(0, -1) : key;
  const i = trimmed.lastIndexOf("/");
  return i >= 0 ? trimmed.slice(i + 1) : trimmed;
}

/** Human-readable byte size, e.g. "1.2 KB". */
function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  const units = ["KB", "MB", "GB", "TB"];
  let value = bytes / 1024;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit += 1;
  }
  return `${value.toFixed(1)} ${units[unit]}`;
}

/** A locale date-time for a modified timestamp, blank when absent/unparseable. */
function formatTime(value?: string): string {
  if (!value) return "—";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? "—" : date.toLocaleString();
}
