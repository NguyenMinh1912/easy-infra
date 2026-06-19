import { Check, Copy } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

interface CopyButtonProps {
  /** Text written to the clipboard when pressed. */
  value: string;
  /** What was copied, named in the toast and the aria-label (e.g. "field"). */
  label?: string;
  className?: string;
}

/**
 * A small icon button that copies `value` to the clipboard, briefly swaps to a
 * check, and toasts. Used for keys and individual hash fields/values where a
 * full edit flow is out of scope but copying is the common need.
 */
export function CopyButton({ value, label = "value", className }: CopyButtonProps) {
  const [copied, setCopied] = useState(false);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      toast.success(`Copied ${label}`);
      window.setTimeout(() => setCopied(false), 1200);
    } catch (cause) {
      toast.error(`Could not copy ${label}`, {
        description: cause instanceof Error ? cause.message : String(cause),
      });
    }
  };

  return (
    <Button
      type="button"
      variant="ghost"
      size="icon"
      onClick={copy}
      aria-label={`Copy ${label}`}
      className={cn("size-7 text-muted-foreground", className)}
    >
      {copied ? <Check className="text-success" aria-hidden /> : <Copy aria-hidden />}
    </Button>
  );
}
