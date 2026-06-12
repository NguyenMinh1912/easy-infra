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

import { baseName, formatBytes, formatTime } from "./format";
import { EntryIcon, PathBreadcrumb } from "./MinioBrowser";
import { ObjectDetailSheet } from "./ObjectDetailSheet";

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
  const [selected, setSelected] = useState<ObjectEntry | null>(null);
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
            onSelect={setSelected}
          />
        ))}
      <ObjectDetailSheet
        profile={profile}
        service={service}
        bucket={bucket}
        object={selected}
        onClose={() => setSelected(null)}
      />
    </div>
  );
}

interface ObjectsTableProps {
  prefixes: string[];
  objects: ObjectEntry[];
  onOpen: (prefix: string) => void;
  onSelect: (object: ObjectEntry) => void;
}

/** The folders and objects of one level, folders first. */
function ObjectsTable({
  prefixes,
  objects,
  onOpen,
  onSelect,
}: ObjectsTableProps) {
  if (prefixes.length === 0 && objects.length === 0) {
    return (
      <div className="rounded-md border border-dashed border-border p-8 text-center text-sm text-muted-foreground">
        This folder is empty.
      </div>
    );
  }
  return (
    <div className="overflow-x-auto rounded-md border border-border">
      <Table className="table-fixed">
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
                  className="flex w-full min-w-0 items-center gap-2 font-medium hover:underline"
                  onClick={() => onOpen(p)}
                >
                  <EntryIcon folder />
                  <span className="truncate" title={`${baseName(p)}/`}>
                    {baseName(p)}/
                  </span>
                </button>
              </TableCell>
              <TableCell className="text-right text-muted-foreground">—</TableCell>
              <TableCell className="text-muted-foreground">—</TableCell>
            </TableRow>
          ))}
          {objects.map((obj) => (
            <TableRow
              key={obj.key}
              className="cursor-pointer"
              onClick={() => onSelect(obj)}
            >
              <TableCell>
                <button
                  type="button"
                  className="flex w-full min-w-0 items-center gap-2 text-left hover:underline"
                  onClick={() => onSelect(obj)}
                >
                  <EntryIcon folder={false} />
                  <span className="truncate font-mono text-sm" title={obj.key}>
                    {baseName(obj.key)}
                  </span>
                </button>
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
