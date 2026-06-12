import { FolderPlus } from "lucide-react";

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

/** Shown on the services screen when the current folder has no project yet. */
export function NoProject() {
  return (
    <Card>
      <CardHeader>
        <div className="mb-2 flex size-10 items-center justify-center rounded-lg bg-muted">
          <FolderPlus className="size-5 text-muted-foreground" />
        </div>
        <CardTitle>No project here</CardTitle>
        <CardDescription>
          There are no services to manage yet.
        </CardDescription>
      </CardHeader>
      <CardContent className="text-sm text-muted-foreground">
        Run{" "}
        <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-foreground">
          easy-infra init
        </code>{" "}
        to scaffold a project, then refresh.
      </CardContent>
    </Card>
  );
}
