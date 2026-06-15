import { Download, Files, Trash2, X } from "lucide-react";
import { useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { cn } from "@/lib/utils";
import { objectsArchiveUrl } from "@/services/api";
import type { ObjectEntry } from "@/types/browse";

import { formatBytes } from "./format";

interface SelectionDetailPanelProps {
  profile: string;
  service: string;
  bucket: string;
  /** Selected objects at the current level. */
  objects: ObjectEntry[];
  /** Selected folder prefixes (each ending in "/"), zipped recursively. */
  prefixes: string[];
  onClose: () => void;
  /** Whether a delete is in flight; disables the delete action while true. */
  deleting?: boolean;
  /** Delete the whole selection after the user confirms. */
  onDelete: () => void;
}

/** A snapshot of the selection, retained so the panel doesn't blank mid-collapse. */
interface Selection {
  objects: ObjectEntry[];
  prefixes: string[];
}

/**
 * The detail panel shown while several items are selected (or a folder is): a
 * summary of the selection with a single action that downloads everything as a
 * zip. It mirrors {@link ObjectDetailPanel}'s slide-in animation — the outer
 * width animates while the inner content keeps a fixed width — and retains the
 * last selection while collapsing so it doesn't blank out mid-animation.
 */
export function SelectionDetailPanel({
  profile,
  service,
  bucket,
  objects,
  prefixes,
  onClose,
  deleting,
  onDelete,
}: SelectionDetailPanelProps) {
  const open = objects.length + prefixes.length > 0;
  const [shown, setShown] = useState<Selection>({ objects, prefixes });
  useEffect(() => {
    if (objects.length + prefixes.length > 0) setShown({ objects, prefixes });
  }, [objects, prefixes]);

  const count = shown.objects.length + shown.prefixes.length;
  const totalBytes = shown.objects.reduce((sum, o) => sum + o.size, 0);

  return (
    <div
      className={cn(
        "shrink-0 self-start overflow-hidden transition-[width] duration-300 ease-in-out",
        open ? "w-[22rem]" : "w-0",
      )}
      aria-hidden={!open}
    >
      <div className="w-[22rem] pl-4">
        <div className="flex flex-col gap-6 rounded-md border border-border bg-card p-5">
          <div className="flex items-start justify-between gap-2">
            <div className="flex min-w-0 flex-col gap-1.5 text-left">
              <h3 className="flex min-w-0 items-center gap-2 text-lg font-semibold">
                <Files
                  className="size-5 shrink-0 text-muted-foreground"
                  aria-hidden
                />
                <span className="truncate">
                  {count} {count === 1 ? "item" : "items"} selected
                </span>
              </h3>
              <p className="text-sm text-muted-foreground">
                Download the selection as a zip
              </p>
            </div>
            <button
              type="button"
              onClick={onClose}
              aria-label="Clear selection"
              className="rounded-sm opacity-70 transition-opacity hover:opacity-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring [&_svg]:size-4"
            >
              <X />
            </button>
          </div>

          <dl className="grid grid-cols-[5rem_1fr] gap-x-4 gap-y-3 text-sm">
            <SummaryRow label="Bucket" value={bucket} />
            <SummaryRow label="Folders" value={String(shown.prefixes.length)} />
            <SummaryRow label="Files" value={String(shown.objects.length)} />
            <SummaryRow
              label="Size"
              value={
                shown.prefixes.length > 0
                  ? `${formatBytes(totalBytes)} + folders`
                  : formatBytes(totalBytes)
              }
            />
          </dl>

          <div className="flex flex-col gap-2">
            <Button asChild className="w-full">
              <a
                href={objectsArchiveUrl(profile, service, bucket, {
                  keys: shown.objects.map((o) => o.key),
                  prefixes: shown.prefixes,
                })}
                download={`${bucket}.zip`}
              >
                <Download aria-hidden />
                Download zip
              </a>
            </Button>
            <ConfirmDialog
              trigger={
                <Button
                  type="button"
                  variant="outline"
                  className="w-full text-destructive hover:text-destructive"
                  disabled={deleting}
                >
                  <Trash2 aria-hidden />
                  Delete
                </Button>
              }
              title={`Delete ${count} ${count === 1 ? "item" : "items"}?`}
              description={
                shown.prefixes.length > 0
                  ? `This permanently removes the selected files and every object inside the selected folders from ${bucket}. This action cannot be undone.`
                  : `This permanently removes the selected files from ${bucket}. This action cannot be undone.`
              }
              confirmLabel="Delete"
              variant="destructive"
              onConfirm={onDelete}
            />
          </div>
        </div>
      </div>
    </div>
  );
}

/** One summary row: a muted label beside its value. */
function SummaryRow({ label, value }: { label: string; value: string }) {
  return (
    <>
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="break-all" title={value}>
        {value}
      </dd>
    </>
  );
}
