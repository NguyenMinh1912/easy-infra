import {
  Boxes,
  Database,
  LayoutDashboard,
  Layers,
  type LucideIcon,
} from "lucide-react";

import { useHashRoute } from "@/hooks/useHashRoute";
import { cn } from "@/lib/utils";

interface NavItem {
  label: string;
  icon: LucideIcon;
  /** Hash route this item links to, e.g. "/services". */
  route?: string;
}

/**
 * Navigation entries for the admin sidebar. Mirrors the tool's domain
 * (dashboard, profiles, services, backup). Items with a `route` are wired to a
 * screen via hash routing; the rest are placeholders for future screens.
 */
const navItems: NavItem[] = [
  { label: "Dashboard", icon: LayoutDashboard, route: "/" },
  { label: "Profiles", icon: Layers },
  { label: "Services", icon: Boxes, route: "/services" },
  { label: "Backup", icon: Database },
];

/** Whether a nav item's route matches the current route. */
function isActive(itemRoute: string, route: string): boolean {
  if (itemRoute === "/") {
    return route === "/" || route === "";
  }
  return route.startsWith(itemRoute);
}

/**
 * Admin sidebar: brand block plus primary navigation. Reads the current hash
 * route to highlight the active item and links each routable entry to its
 * screen.
 */
export function Sidebar() {
  const route = useHashRoute();

  return (
    <aside className="flex w-60 shrink-0 flex-col border-r bg-card">
      <div className="flex h-16 items-center border-b px-6">
        <span className="text-lg font-bold tracking-tight">easy-infra</span>
      </div>
      <nav className="flex-1 space-y-1 p-4">
        {navItems.map(({ label, icon: Icon, route: itemRoute }) => {
          const active = itemRoute !== undefined && isActive(itemRoute, route);
          return (
            <a
              key={label}
              href={itemRoute ? `#${itemRoute}` : "#"}
              aria-current={active ? "page" : undefined}
              className={cn(
                "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
                active
                  ? "bg-accent text-accent-foreground"
                  : "text-muted-foreground hover:bg-accent hover:text-accent-foreground",
              )}
            >
              <Icon className="size-4 shrink-0" />
              {label}
            </a>
          );
        })}
      </nav>
    </aside>
  );
}
