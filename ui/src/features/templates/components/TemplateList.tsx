import { Pencil, Trash2 } from "lucide-react";

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
import type { TemplateSummary } from "@/types/templates";

interface TemplateListProps {
  templates: TemplateSummary[];
  onEdit: (name: string) => void;
  onDelete: (name: string) => void;
}

/**
 * The workspace's SQL templates, one row each: name, description, variable
 * count, last-updated, and edit/delete actions. Deleting confirms first.
 * Templates are run from the SQL console via the "/" mention menu.
 */
export function TemplateList({
  templates,
  onEdit,
  onDelete,
}: TemplateListProps) {
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Name</TableHead>
          <TableHead>Description</TableHead>
          <TableHead>Variables</TableHead>
          <TableHead>Updated</TableHead>
          <TableHead className="text-right">Actions</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {templates.map((t) => (
          <TableRow key={t.name}>
            <TableCell className="font-medium">{t.name}</TableCell>
            <TableCell className="text-muted-foreground">
              {t.description || "—"}
            </TableCell>
            <TableCell>
              {t.variables.length > 0 ? (
                <Badge variant="secondary">{t.variables.length}</Badge>
              ) : (
                <span className="text-muted-foreground">0</span>
              )}
            </TableCell>
            <TableCell className="text-muted-foreground">
              {formatWhen(t.updatedAt)}
            </TableCell>
            <TableCell className="text-right">
              <div className="flex justify-end gap-1">
                <Button
                  size="icon"
                  variant="ghost"
                  className="size-8"
                  aria-label={`Edit ${t.name}`}
                  onClick={() => onEdit(t.name)}
                >
                  <Pencil aria-hidden />
                </Button>
                <ConfirmDialog
                  trigger={
                    <Button
                      size="icon"
                      variant="ghost"
                      className="size-8 text-muted-foreground hover:text-destructive"
                      aria-label={`Delete ${t.name}`}
                    >
                      <Trash2 aria-hidden />
                    </Button>
                  }
                  title={`Delete template "${t.name}"?`}
                  description="This permanently removes the saved SQL script. This cannot be undone."
                  confirmLabel="Delete"
                  variant="destructive"
                  onConfirm={() => onDelete(t.name)}
                />
              </div>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

/** Render an RFC 3339 timestamp as a short local date-time, or "—" if absent. */
function formatWhen(iso: string): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "—";
  return d.toLocaleString();
}
