import { useState } from "react";
import { AlertCircle, FileCode, Plus } from "lucide-react";
import { toast } from "sonner";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { ApiError, deleteTemplate } from "@/services/api";

import { TemplateEditor } from "./components/TemplateEditor";
import { TemplateList } from "./components/TemplateList";
import { useTemplates } from "./hooks/useTemplates";

/**
 * Container for the SQL Templates screen: owns data loading via
 * {@link useTemplates} and the create/edit dialogs, mapping each async state to
 * a view. The only place wiring data to view. Templates are run from the SQL
 * console via its "/" mention menu, not from this screen.
 */
export function TemplatesPage() {
  const { state, reload } = useTemplates();

  // Editor: editName undefined = create, a string = edit that template.
  const [editorOpen, setEditorOpen] = useState(false);
  const [editName, setEditName] = useState<string | undefined>(undefined);

  const openCreate = () => {
    setEditName(undefined);
    setEditorOpen(true);
  };
  const openEdit = (name: string) => {
    setEditName(name);
    setEditorOpen(true);
  };
  const remove = async (name: string) => {
    try {
      await deleteTemplate(name);
      toast.success(`Deleted "${name}"`);
      reload();
    } catch (cause) {
      toast.error("Delete failed", {
        description: cause instanceof ApiError ? cause.message : String(cause),
      });
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex justify-end">
        <Button onClick={openCreate}>
          <Plus aria-hidden />
          New Template
        </Button>
      </div>

      {state.status === "loading" && (
        <div className="space-y-2">
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
        </div>
      )}

      {state.status === "error" && (
        <Alert variant="destructive">
          <AlertCircle />
          <div>
            <AlertTitle>Could not reach the API</AlertTitle>
            <AlertDescription>
              {state.error.message}. Make sure <code>easy-infra serve</code> is
              running, then refresh.
            </AlertDescription>
          </div>
        </Alert>
      )}

      {state.status === "success" &&
        (state.data.length === 0 ? (
          <Card>
            <CardHeader>
              <div className="mb-2 flex size-10 items-center justify-center rounded-lg bg-muted">
                <FileCode className="size-5 text-muted-foreground" aria-hidden />
              </div>
              <CardTitle>No templates yet</CardTitle>
              <CardDescription>
                Create one to save a reusable, parameterized SQL script.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <Button onClick={openCreate}>
                <Plus aria-hidden />
                New Template
              </Button>
            </CardContent>
          </Card>
        ) : (
          <TemplateList
            templates={state.data}
            onEdit={openEdit}
            onDelete={(name) => void remove(name)}
          />
        ))}

      <TemplateEditor
        open={editorOpen}
        onOpenChange={setEditorOpen}
        editName={editName}
        onSaved={reload}
      />
    </div>
  );
}
