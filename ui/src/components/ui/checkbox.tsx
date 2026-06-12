import * as React from "react";
import { Check, Minus } from "lucide-react";

import { cn } from "@/lib/utils";

interface CheckboxProps {
  checked: boolean;
  /** Renders the mixed (partial) state, e.g. a "select all" with some checked. */
  indeterminate?: boolean;
  onCheckedChange: () => void;
  "aria-label"?: string;
  className?: string;
}

/**
 * A small tri-state checkbox: a button with `role="checkbox"`, matching the
 * codebase's existing aria-checked toggle pattern. Indeterminate shows a dash
 * and reports `aria-checked="mixed"`.
 */
export const Checkbox = React.forwardRef<HTMLButtonElement, CheckboxProps>(
  ({ checked, indeterminate, onCheckedChange, className, ...props }, ref) => (
    <button
      ref={ref}
      type="button"
      role="checkbox"
      aria-checked={indeterminate ? "mixed" : checked}
      onClick={onCheckedChange}
      className={cn(
        "flex size-4 shrink-0 items-center justify-center rounded-sm border border-input transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
        (checked || indeterminate) &&
          "border-primary bg-primary text-primary-foreground",
        className,
      )}
      {...props}
    >
      {indeterminate ? (
        <Minus className="size-3" aria-hidden />
      ) : checked ? (
        <Check className="size-3" aria-hidden />
      ) : null}
    </button>
  ),
);
Checkbox.displayName = "Checkbox";
