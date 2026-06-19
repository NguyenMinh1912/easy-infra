import { ChevronsUpDown, Database } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

interface DbSelectorProps {
  /** Currently selected logical database. */
  db: number;
  /** Number of logical databases to offer (at least 1). */
  count: number;
  onChange: (db: number) => void;
}

/**
 * Logical-database picker shared by the Redis key browser and console, so both
 * surfaces select a database the same way and the console can show which one
 * its commands run against.
 */
export function DbSelector({ db, count, onChange }: DbSelectorProps) {
  const databases = Math.max(1, count);
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="justify-between gap-2"
          aria-label="Select database"
        >
          <span className="flex items-center gap-2">
            <Database className="size-4 text-muted-foreground" aria-hidden />
            db {db}
          </span>
          <ChevronsUpDown className="size-4 text-muted-foreground" aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="max-h-72 overflow-y-auto">
        {Array.from({ length: databases }, (_, i) => (
          <DropdownMenuItem key={i} onSelect={() => onChange(i)}>
            db {i}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
