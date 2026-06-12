import type { ReactNode } from "react";

interface ServiceDetailLayoutProps {
  /** Action controls rendered at the right of the navbar (e.g. the action menu). */
  actions?: ReactNode;
  /** The service content shown below the navbar. */
  children: ReactNode;
}

/**
 * Shared chrome for every service detail page: a navbar with the service action
 * menu, above the service content. The content is service-specific and passed in
 * as children, so each service reuses the same layout and only the panel below
 * differs.
 */
export function ServiceDetailLayout({
  actions,
  children,
}: ServiceDetailLayoutProps) {
  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-end gap-3 border-b border-border pb-3">
        {actions}
      </div>
      <div>{children}</div>
    </div>
  );
}
