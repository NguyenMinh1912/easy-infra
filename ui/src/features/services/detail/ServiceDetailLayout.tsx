import { useState, type ReactNode } from "react";
import { ArrowLeft, type LucideIcon } from "lucide-react";

import { cn } from "@/lib/utils";

/** One selectable section of a service detail page. */
export interface ServiceSection {
  id: string;
  label: string;
  icon: LucideIcon;
  content: ReactNode;
}

interface ServiceDetailLayoutProps {
  sections: ServiceSection[];
}

/**
 * Shared chrome for every service detail page: a control navbar (section tabs
 * plus a link back to the services list) above the active section's content.
 * Service-specific views are embedded as section content, so each service
 * reuses the same layout and only the panels differ. Owns which section is
 * active.
 */
export function ServiceDetailLayout({ sections }: ServiceDetailLayoutProps) {
  const [activeId, setActiveId] = useState(sections[0]?.id);
  const active = sections.find((s) => s.id === activeId) ?? sections[0];

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3 border-b border-border pb-3">
        <nav className="flex items-center gap-1" aria-label="Service sections">
          {sections.map((section) => {
            const Icon = section.icon;
            const isActive = section.id === active?.id;
            return (
              <button
                key={section.id}
                type="button"
                onClick={() => setActiveId(section.id)}
                aria-current={isActive ? "page" : undefined}
                className={cn(
                  "flex items-center gap-2 rounded-md px-3 py-1.5 text-sm font-medium transition-colors",
                  isActive
                    ? "bg-accent text-accent-foreground"
                    : "text-muted-foreground hover:bg-accent hover:text-accent-foreground",
                )}
              >
                <Icon className="size-4 shrink-0" aria-hidden />
                {section.label}
              </button>
            );
          })}
        </nav>
        <a
          href="#/services"
          className="flex items-center gap-1.5 rounded-md px-2 py-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground"
        >
          <ArrowLeft className="size-4 shrink-0" aria-hidden />
          All services
        </a>
      </div>
      <div>{active?.content}</div>
    </div>
  );
}
