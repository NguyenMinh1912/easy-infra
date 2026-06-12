import { AlertCircle } from "lucide-react";
import { useEffect, useState } from "react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Checkbox } from "@/components/ui/checkbox";
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
import { ObjectDetailPanel } from "./ObjectDetailPanel";
import { SelectionDetailPanel } from "./SelectionDetailPanel";

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
 *
 * Rows carry selection checkboxes. Clicking a row keeps the single-object
 * detail behaviour; checking boxes drives a multi-select summary that downloads
 * the selection as a zip — except a lone selected object (no folders), which
 * falls back to the same single-object detail.
 */
export function BucketContents({
  profile,
  service,
  bucket,
}: BucketContentsProps) {
  const [prefix, setPrefix] = useState("");
  const [selected, setSelected] = useState<ObjectEntry | null>(null);
  const [checked, setChecked] = useState<Set<string>>(new Set());
  const state = useAsync(
    (signal) => listObjects(profile, service, bucket, prefix, signal),
    [profile, service, bucket, prefix],
  );

  // Selection is scoped to one folder level; reset it on navigation.
  useEffect(() => {
    setSelected(null);
    setChecked(new Set());
  }, [prefix]);

  const listing =
    state.status === "success" && !state.data.error ? state.data : null;
  const prefixes = listing?.prefixes ?? [];
  const objects = listing?.objects ?? [];

  // Row click opens the single-object detail; toggling a checkbox switches to
  // multi-select. The two are mutually exclusive, so each clears the other.
  const openObject = (object: ObjectEntry) => {
    setChecked(new Set());
    setSelected(object);
  };
  const toggle = (id: string) => {
    setSelected(null);
    setChecked((current) => {
      const next = new Set(current);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };
  const ids = [...prefixes, ...objects.map((o) => o.key)];
  const allChecked = ids.length > 0 && ids.every((id) => checked.has(id));
  const someChecked = ids.some((id) => checked.has(id));
  const toggleAll = () => {
    setSelected(null);
    setChecked(allChecked ? new Set() : new Set(ids));
  };

  // Decide which side panel to show. A single selected object with no folders
  // keeps the single-object detail; any larger selection shows the zip summary.
  const checkedObjects = objects.filter((o) => checked.has(o.key));
  const checkedPrefixes = prefixes.filter((p) => checked.has(p));
  const loneObject =
    checkedObjects.length === 1 && checkedPrefixes.length === 0
      ? checkedObjects[0]
      : null;
  const showSelection = checked.size > 0 && loneObject === null;
  const detailObject = showSelection ? null : (selected ?? loneObject);
  const clearSelection = () => {
    setSelected(null);
    setChecked(new Set());
  };

  return (
    <div className="space-y-3">
      <PathBreadcrumb bucket={bucket} prefix={prefix} onNavigate={setPrefix} />
      <div className="flex items-start">
        <div className="min-w-0 flex-1">
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
                prefixes={prefixes}
                objects={objects}
                checked={checked}
                allChecked={allChecked}
                someChecked={someChecked}
                onOpen={setPrefix}
                onSelect={openObject}
                onToggle={toggle}
                onToggleAll={toggleAll}
              />
            ))}
        </div>
        <ObjectDetailPanel
          profile={profile}
          service={service}
          bucket={bucket}
          object={detailObject}
          onClose={clearSelection}
        />
        <SelectionDetailPanel
          profile={profile}
          service={service}
          bucket={bucket}
          objects={showSelection ? checkedObjects : []}
          prefixes={showSelection ? checkedPrefixes : []}
          onClose={clearSelection}
        />
      </div>
    </div>
  );
}

interface ObjectsTableProps {
  prefixes: string[];
  objects: ObjectEntry[];
  checked: Set<string>;
  allChecked: boolean;
  someChecked: boolean;
  onOpen: (prefix: string) => void;
  onSelect: (object: ObjectEntry) => void;
  onToggle: (id: string) => void;
  onToggleAll: () => void;
}

/** The folders and objects of one level, folders first. */
function ObjectsTable({
  prefixes,
  objects,
  checked,
  allChecked,
  someChecked,
  onOpen,
  onSelect,
  onToggle,
  onToggleAll,
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
            <TableHead className="w-10">
              <Checkbox
                checked={allChecked}
                indeterminate={someChecked && !allChecked}
                onCheckedChange={onToggleAll}
                aria-label="Select all"
              />
            </TableHead>
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
              <SelectCell
                checked={checked.has(p)}
                onToggle={() => onToggle(p)}
                label={`Select ${baseName(p)}/`}
              />
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
              <SelectCell
                checked={checked.has(obj.key)}
                onToggle={() => onToggle(obj.key)}
                label={`Select ${baseName(obj.key)}`}
              />
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

/**
 * The leading checkbox cell of a row. It swallows the click so toggling the
 * selection doesn't also trigger the row's navigate/open behaviour.
 */
function SelectCell({
  checked,
  onToggle,
  label,
}: {
  checked: boolean;
  onToggle: () => void;
  label: string;
}) {
  return (
    <TableCell className="w-10" onClick={(e) => e.stopPropagation()}>
      <Checkbox checked={checked} onCheckedChange={onToggle} aria-label={label} />
    </TableCell>
  );
}
