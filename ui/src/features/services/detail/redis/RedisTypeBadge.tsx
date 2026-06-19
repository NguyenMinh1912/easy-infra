import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

/**
 * Per-type tints for Redis value types. Gray badges all look alike, so each
 * type gets a distinct, low-saturation colour that reads in both themes — the
 * type is then scannable at a glance in the key list and the value header.
 * Unknown types (e.g. stream) fall back to a neutral tint.
 */
const TYPE_STYLES: Record<string, string> = {
  string: "border-blue-500/25 bg-blue-500/10 text-blue-700 dark:text-blue-300",
  hash: "border-violet-500/25 bg-violet-500/10 text-violet-700 dark:text-violet-300",
  list: "border-amber-500/25 bg-amber-500/10 text-amber-700 dark:text-amber-300",
  set: "border-emerald-500/25 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300",
  zset: "border-rose-500/25 bg-rose-500/10 text-rose-700 dark:text-rose-300",
  stream: "border-cyan-500/25 bg-cyan-500/10 text-cyan-700 dark:text-cyan-300",
};

const FALLBACK = "border-border bg-muted text-muted-foreground";

/** A colour-coded badge naming a Redis value type. */
export function RedisTypeBadge({
  type,
  className,
}: {
  type: string;
  className?: string;
}) {
  return (
    <Badge
      variant="outline"
      className={cn(TYPE_STYLES[type] ?? FALLBACK, className)}
    >
      {type}
    </Badge>
  );
}
