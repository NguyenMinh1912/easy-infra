import { useCallback, useState } from "react";
import { ChevronLeft, ChevronRight, Database } from "lucide-react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { deleteBackup, type BackupList, type BackupSession } from "@/services/api";

import { BackupsTable } from "./BackupsTable";
import { BackupSessionDialog } from "./BackupSessionDialog";

interface BackupsManagerProps {
  data: BackupList;
  page: number;
  pageSize: number;
  setPage: (page: number) => void;
  reload: () => void;
}

/**
 * Presentation + wiring for the backups screen: the sessions table, pagination
 * controls, the per-session view dialog, and delete. Deleting toasts the
 * outcome and refreshes the page; viewing opens the log dialog (which polls a
 * still-running session).
 */
export function BackupsManager({
  data,
  page,
  pageSize,
  setPage,
  reload,
}: BackupsManagerProps) {
  const [busy, setBusy] = useState(false);
  const [viewing, setViewing] = useState<BackupSession | null>(null);

  const totalPages = Math.max(1, Math.ceil(data.total / pageSize));

  const remove = useCallback(
    async (session: BackupSession) => {
      setBusy(true);
      try {
        await deleteBackup(session.id);
        toast.success("Backup deleted");
        // If we just removed the last row of a non-first page, step back so the
        // user does not land on an empty page.
        if (data.sessions.length === 1 && page > 1) {
          setPage(page - 1);
        } else {
          reload();
        }
      } catch (cause) {
        toast.error("Could not delete backup", {
          description: cause instanceof Error ? cause.message : String(cause),
        });
      } finally {
        setBusy(false);
      }
    },
    [data.sessions.length, page, setPage, reload],
  );

  // Reopen-friendly: closing the dialog refreshes the list so a backup that
  // finished while open shows its final status.
  const closeDialog = useCallback(() => {
    setViewing(null);
    reload();
  }, [reload]);

  if (data.sessions.length === 0) {
    return (
      <Card className="flex flex-col items-center gap-3 p-10 text-center">
        <span className="flex size-10 items-center justify-center rounded-lg bg-muted">
          <Database className="size-5 text-muted-foreground" aria-hidden />
        </span>
        <div className="space-y-1">
          <p className="font-medium">No backups yet</p>
          <p className="text-sm text-muted-foreground">
            Run a backup from a service's detail page; sessions show up here.
          </p>
        </div>
      </Card>
    );
  }

  return (
    <div className="space-y-4">
      <p className="text-sm text-muted-foreground">
        {data.total} backup{data.total === 1 ? "" : "s"}
      </p>

      <Card>
        <BackupsTable
          sessions={data.sessions}
          busy={busy}
          onView={setViewing}
          onDelete={remove}
        />
      </Card>

      {totalPages > 1 && (
        <div className="flex items-center justify-between">
          <p className="text-sm text-muted-foreground">
            Page {page} of {totalPages}
          </p>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={busy || page <= 1}
              onClick={() => setPage(page - 1)}
            >
              <ChevronLeft />
              Previous
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={busy || page >= totalPages}
              onClick={() => setPage(page + 1)}
            >
              Next
              <ChevronRight />
            </Button>
          </div>
        </div>
      )}

      <BackupSessionDialog
        session={viewing}
        onOpenChange={(open) => {
          if (!open) closeDialog();
        }}
      />
    </div>
  );
}
