import { FolderPlus } from "lucide-react";

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

/** Shown when the current folder has no easy-infra project yet. */
export function EmptyProject() {
  return (
    <Card>
      <CardHeader>
        <div className="mb-2 flex size-10 items-center justify-center rounded-lg bg-muted">
          <FolderPlus className="size-5 text-muted-foreground" />
        </div>
        <CardTitle>No project here</CardTitle>
        <CardDescription>
          This folder has no easy-infra project yet.
        </CardDescription>
      </CardHeader>
      <CardContent className="text-sm text-muted-foreground">
        Run{" "}
        <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-foreground">
          easy-infra init
        </code>{" "}
        to scaffold one, then refresh.
      </CardContent>
    </Card>
  );
}
