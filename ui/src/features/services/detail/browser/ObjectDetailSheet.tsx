import { Download, File } from "lucide-react";
import { useEffect, useState } from "react";

import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { objectDownloadUrl } from "@/services/api";
import type { ObjectEntry } from "@/types/browse";

import { baseName, formatBytes, formatTime } from "./format";

interface ObjectDetailSheetProps {
  profile: string;
  service: string;
  bucket: string;
  /** The object whose detail is shown, or null when the panel is closed. */
  object: ObjectEntry | null;
  onClose: () => void;
}

/**
 * A right-anchored panel that slides in to show one object's metadata and a
 * download action. Driven by `object`: non-null opens it, null closes it. The
 * last shown object is retained while the close animation plays so the panel
 * doesn't blank out mid-slide.
 */
export function ObjectDetailSheet({
  profile,
  service,
  bucket,
  object,
  onClose,
}: ObjectDetailSheetProps) {
  const [shown, setShown] = useState<ObjectEntry | null>(object);
  useEffect(() => {
    if (object) setShown(object);
  }, [object]);

  return (
    <Sheet
      open={object !== null}
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
    >
      <SheetContent side="right" className="gap-6">
        {shown && (
          <>
            <SheetHeader>
              <SheetTitle className="flex min-w-0 items-center gap-2">
                <File
                  className="size-5 shrink-0 text-muted-foreground"
                  aria-hidden
                />
                <span className="truncate" title={baseName(shown.key)}>
                  {baseName(shown.key)}
                </span>
              </SheetTitle>
              <SheetDescription>Object details</SheetDescription>
            </SheetHeader>

            <dl className="grid grid-cols-[6rem_1fr] gap-x-4 gap-y-3 text-sm">
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

            <SheetFooter>
              <Button asChild>
                <a
                  href={objectDownloadUrl(profile, service, bucket, shown.key)}
                  download={baseName(shown.key)}
                >
                  <Download aria-hidden />
                  Download
                </a>
              </Button>
            </SheetFooter>
          </>
        )}
      </SheetContent>
    </Sheet>
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
