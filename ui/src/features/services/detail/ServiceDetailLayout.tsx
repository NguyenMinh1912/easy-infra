import type { ReactNode } from "react";
import { ArrowLeft } from "lucide-react";

interface ServiceDetailLayoutProps {
  /** Action controls rendered at the right of the navbar (e.g. the action menu). */
  actions?: ReactNode;
  /** The service content shown below the navbar. */
  children: ReactNode;
}

/**
 * Shared chrome for every service detail page: a navbar with a link back to the
 * services list and the service action menu, above the service content. The
 * content is service-specific and passed in as children, so each service reuses
 * the same layout and only the panel below differs.
 */
export function ServiceDetailLayout({
  actions,
  children,
}: ServiceDetailLayoutProps) {
  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3 border-b border-border pb-3">
        <a
          href="#/services"
          className="flex items-center gap-1.5 rounded-md px-2 py-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground"
        >
          <ArrowLeft className="size-4 shrink-0" aria-hidden />
          All services
        </a>
        {actions}
      </div>
      <div>{children}</div>
    </div>
  );
}
