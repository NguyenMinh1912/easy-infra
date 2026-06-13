import { useCallback, useRef, useState } from "react";
import { toast } from "sonner";

import { uploadObject } from "@/services/api";

/**
 * Upload at most this many files at once. A bounded pool keeps the page
 * responsive when a large multi-file selection (or folder drop) is queued: the
 * remaining files wait rather than opening hundreds of simultaneous requests.
 */
const MAX_CONCURRENT = 3;

/** Live progress of an in-flight batch upload. */
export interface UploadProgress {
  total: number;
  done: number;
  failed: number;
}

interface UseObjectUploadParams {
  profile: string;
  service: string;
  bucket: string;
  /** Folder prefix the files land under (ends in "/" or is empty for root). */
  prefix: string;
  /** Called once a batch finishes so the listing can reload. */
  onComplete: () => void;
}

/**
 * Upload files into the current bucket folder with bounded concurrency. Each
 * file streams straight to the store as the request body (never buffered whole
 * in JS), and only {@link MAX_CONCURRENT} run at a time. Exposes the live
 * {@link UploadProgress} for a status line and reports the outcome via a toast.
 */
export function useObjectUpload({
  profile,
  service,
  bucket,
  prefix,
  onComplete,
}: UseObjectUploadParams) {
  const [progress, setProgress] = useState<UploadProgress | null>(null);
  // Guards against starting a second batch while one is already running.
  const running = useRef(false);

  const start = useCallback(
    async (files: File[]) => {
      if (files.length === 0 || running.current) return;
      running.current = true;

      let done = 0;
      let failed = 0;
      const failures: string[] = [];
      setProgress({ total: files.length, done, failed });

      // Pull from a shared cursor so each worker grabs the next pending file;
      // single-threaded JS makes the increment race-free.
      let next = 0;
      const worker = async () => {
        while (next < files.length) {
          const file = files[next++];
          try {
            await uploadObject(
              profile,
              service,
              bucket,
              `${prefix}${file.name}`,
              file,
            );
            done++;
          } catch {
            failed++;
            failures.push(file.name);
          }
          setProgress({ total: files.length, done, failed });
        }
      };
      await Promise.all(
        Array.from({ length: Math.min(MAX_CONCURRENT, files.length) }, worker),
      );

      running.current = false;
      setProgress(null);
      if (failed === 0) {
        toast.success(`Uploaded ${done} ${done === 1 ? "file" : "files"}`);
      } else {
        toast.error(
          `${failed} of ${files.length} ${files.length === 1 ? "upload" : "uploads"} failed`,
          { description: failures.slice(0, 5).join(", ") },
        );
      }
      onComplete();
    },
    [profile, service, bucket, prefix, onComplete],
  );

  return { progress, uploading: progress !== null, start };
}
