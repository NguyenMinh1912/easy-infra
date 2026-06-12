import { Eye, Trash2 } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { cn } from "@/lib/utils";
import { metaFor } from "@/features/services";
import type { BackupSession } from "@/services/api";

import { KIND_META } from "../kind-meta";
import { STATUS_META } from "../status-meta";

interface BackupsTableProps {
  sessions: BackupSession[];
  busy: boolean;
  onView: (session: BackupSession) => void;
  onDelete: (session: BackupSession) => void;
}

/**
 * The backup sessions, one row each: service identity, profile, a status badge,
 * when it was created, and view/delete actions. Clicking a row (or the eye
 * icon) opens the session dialog; running sessions cannot be deleted, so their
 * delete action is disabled with a hint to cancel first.
 */
export function BackupsTable({
  sessions,
  busy,
  onView,
  onDelete,
}: BackupsTableProps) {
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Service</TableHead>
          <TableHead>Profile</TableHead>
          <TableHead>Type</TableHead>
          <TableHead>Status</TableHead>
          <TableHead>Created</TableHead>
          <TableHead className="text-right">Actions</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {sessions.map((session) => {
          const svc = metaFor(session.service);
          const SvcIcon = svc.icon;
          const kind = KIND_META[session.kind];
          const KindIcon = kind.icon;
          const status = STATUS_META[session.status];
          const StatusIcon = status.icon;
          const running = session.status === "running";
          return (
            <TableRow
              key={session.id}
              className="cursor-pointer"
              onClick={() => onView(session)}
            >
              <TableCell>
                <div className="flex items-center gap-3">
                  <span className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-muted">
                    <SvcIcon
                      className="size-4 text-muted-foreground"
                      aria-hidden
                    />
                  </span>
                  <span>
                    <span className="font-medium">{svc.label}</span>
                    <span className="block font-mono text-xs text-muted-foreground">
                      {session.service}
                    </span>
                  </span>
                </div>
              </TableCell>
              <TableCell className="font-mono text-xs text-muted-foreground">
                {session.profile}
              </TableCell>
              <TableCell>
                <Badge variant={kind.variant}>
                  <KindIcon className="size-3" aria-hidden />
                  {kind.label}
                </Badge>
              </TableCell>
              <TableCell>
                <Badge variant={status.variant}>
                  <StatusIcon
                    className={cn("size-3", status.spin && "animate-spin")}
                    aria-hidden
                  />
                  {status.label}
                </Badge>
              </TableCell>
              <TableCell className="text-sm text-muted-foreground">
                {formatTime(session.createdAt)}
              </TableCell>
              <TableCell onClick={(e) => e.stopPropagation()}>
                <div className="flex items-center justify-end gap-1">
                  <Button
                    variant="ghost"
                    size="icon"
                    aria-label={`View ${session.service} backup`}
                    onClick={() => onView(session)}
                  >
                    <Eye />
                  </Button>
                  <ConfirmDialog
                    trigger={
                      <Button
                        variant="ghost"
                        size="icon"
                        aria-label={`Delete ${session.service} backup`}
                        disabled={busy || running}
                        title={
                          running
                            ? "Cancel the running backup before deleting it"
                            : undefined
                        }
                      >
                        <Trash2 />
                      </Button>
                    }
                    title="Delete this backup?"
                    description="This removes the session, its log, and the snapshot on disk. This action cannot be undone."
                    confirmLabel="Delete"
                    variant="destructive"
                    onConfirm={() => onDelete(session)}
                  />
                </div>
              </TableCell>
            </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );
}

/** Format an ISO timestamp for display, falling back to the raw string. */
function formatTime(iso: string): string {
  const date = new Date(iso);
  return Number.isNaN(date.getTime()) ? iso : date.toLocaleString();
}
