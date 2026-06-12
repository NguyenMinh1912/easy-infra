import { Layers } from "lucide-react";

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

interface ActiveProfileCardProps {
  activeProfile: string;
}

/** Highlights the currently active profile. */
export function ActiveProfileCard({ activeProfile }: ActiveProfileCardProps) {
  return (
    <Card>
      <CardHeader className="flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">
          Active profile
        </CardTitle>
        <Layers className="size-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <p className="text-2xl font-semibold">
          {activeProfile || (
            <span className="text-muted-foreground">— none —</span>
          )}
        </p>
      </CardContent>
    </Card>
  );
}
