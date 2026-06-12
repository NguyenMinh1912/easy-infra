import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";

/** Loading placeholder mirroring the backups screen layout (count + table). */
export function BackupsSkeleton() {
  return (
    <div className="space-y-4">
      <Skeleton className="h-4 w-24" />
      <Card className="divide-y p-0">
        {[0, 1, 2].map((i) => (
          <div key={i} className="flex items-center gap-3 p-4">
            <Skeleton className="size-9 shrink-0 rounded-lg" />
            <div className="flex-1 space-y-2">
              <Skeleton className="h-4 w-32" />
              <Skeleton className="h-3 w-48" />
            </div>
            <Skeleton className="h-6 w-20 rounded-full" />
            <Skeleton className="h-8 w-16" />
          </div>
        ))}
      </Card>
    </div>
  );
}
