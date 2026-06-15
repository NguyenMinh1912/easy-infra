import { Download, File, Trash2, X } from "lucide-react";
import { useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { cn } from "@/lib/utils";
import { objectDownloadUrl } from "@/services/api";
import type { ObjectEntry } from "@/types/browse";

import { baseName, formatBytes, formatTime } from "./format";

interface ObjectDetailPanelProps {
  profile: string;
  service: string;
  bucket: string;
  /** The object whose detail is shown, or null when the panel is collapsed. */
  object: ObjectEntry | null;
  onClose: () => void;
  /** Whether a delete is in flight; disables the delete action while true. */
  deleting?: boolean;
  /** Delete the shown object after the user confirms. */
  onDelete: () => void;
}

/**
 * An inline detail panel that lives beside the object table within the service
 * content (not a viewport overlay). Driven by `object`: non-null expands it,
 * null collapses its width back to zero. The animation runs on the outer
 * width while the inner content keeps a fixed width, so it slides rather than
 * reflows. The last shown object is retained while collapsing so the panel
 * doesn't blank out mid-animation.
 */
export function ObjectDetailPanel({
  profile,
  service,
  bucket,
  object,
  onClose,
  deleting,
  onDelete,
}: ObjectDetailPanelProps) {
  const [shown, setShown] = useState<ObjectEntry | null>(object);
  useEffect(() => {
    if (object) setShown(object);
  }, [object]);

  const open = object !== null;

  return (
    <div
      className={cn(
        "shrink-0 self-start overflow-hidden transition-[width] duration-300 ease-in-out",
        open ? "w-[22rem]" : "w-0",
      )}
      aria-hidden={!open}
    >
      {/* Fixed width keeps the content stable while the outer width animates;
          the left padding is the gap from the table and is clipped when closed. */}
      <div className="w-[22rem] pl-4">
        {shown && (
          <div className="flex flex-col gap-6 rounded-md border border-border bg-card p-5">
            <div className="flex items-start justify-between gap-2">
              <div className="flex min-w-0 flex-col gap-1.5 text-left">
                <h3 className="flex min-w-0 items-center gap-2 text-lg font-semibold">
                  <File
                    className="size-5 shrink-0 text-muted-foreground"
                    aria-hidden
                  />
                  <span className="truncate" title={baseName(shown.key)}>
                    {baseName(shown.key)}
                  </span>
                </h3>
                <p className="text-sm text-muted-foreground">Object details</p>
              </div>
              <button
                type="button"
                onClick={onClose}
                aria-label="Close"
                className="rounded-sm opacity-70 transition-opacity hover:opacity-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring [&_svg]:size-4"
              >
                <X />
              </button>
            </div>

            <dl className="grid grid-cols-[5rem_1fr] gap-x-4 gap-y-3 text-sm">
              <DetailRow label="Bucket" value={bucket} />
              <DetailRow label="Key" value={shown.key} mono />
              <DetailRow label="Size" value={formatBytes(shown.size)} mono />
              <DetailRow
                label="Type"
                value={shown.contentType || "—"}
                mono={Boolean(shown.contentType)}
              />
              <DetailRow label="Modified" value={formatTime(shown.lastModified)} />
            </dl>

            <div className="flex flex-col gap-2">
              <Button asChild className="w-full">
                <a
                  href={objectDownloadUrl(profile, service, bucket, shown.key)}
                  download={baseName(shown.key)}
                >
                  <Download aria-hidden />
                  Download
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
                title="Delete this object?"
                description={
                  <>
                    This permanently removes{" "}
                    <span className="font-mono">{baseName(shown.key)}</span> from{" "}
                    {bucket}. This action cannot be undone.
                  </>
                }
                confirmLabel="Delete"
                variant="destructive"
                onConfirm={onDelete}
              />
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

/** One metadata row: a muted label beside its value. */
function DetailRow({
  label,
  value,
  mono,
}: {
  label: string;
  value: string;
  mono?: boolean;
}) {
  return (
    <>
      <dt className="text-muted-foreground">{label}</dt>
      <dd className={mono ? "break-all font-mono" : "break-all"} title={value}>
        {value}
      </dd>
    </>
  );
}
