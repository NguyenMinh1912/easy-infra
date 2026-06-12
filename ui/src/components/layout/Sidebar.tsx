import {
  Boxes,
  Database,
  LayoutDashboard,
  Server,
  type LucideIcon,
} from "lucide-react";

import { ThemeToggle } from "@/components/theme/ThemeToggle";
import { ProfileNav } from "@/features/profiles";
import { useHashRoute } from "@/hooks/useHashRoute";
import { cn } from "@/lib/utils";

interface NavItem {
  label: string;
  icon: LucideIcon;
  /** Hash route this item links to, e.g. "/services". */
  route?: string;
}

/**
 * Top-level navigation entries for the admin sidebar. Profiles is rendered
 * separately by {@link ProfileNav} as an expandable section, so it is not
 * listed here. Items with a `route` are wired to a screen via hash routing;
 * the rest are placeholders for future screens.
 */
const navItems: NavItem[] = [
  { label: "Dashboard", icon: LayoutDashboard, route: "/" },
  { label: "Services", icon: Boxes, route: "/services" },
  { label: "Backups", icon: Database, route: "/backups" },
];

/**
 * Whether a nav item's route matches the current route. Matches exactly so a
 * service detail page (`/services/{name}`), which belongs to a profile rather
 * than the top-level Services list, does not highlight the Services item.
 */
function isActive(itemRoute: string, route: string): boolean {
  if (itemRoute === "/") {
    return route === "/" || route === "";
  }
  return route === itemRoute;
}

/** A single top-level sidebar link, highlighted when its route is active. */
function NavLink({ item, route }: { item: NavItem; route: string }) {
  const { label, icon: Icon, route: itemRoute } = item;
  const active = itemRoute !== undefined && isActive(itemRoute, route);
  return (
    <a
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
      <div className="flex h-16 items-center gap-2.5 border-b px-6">
        <span className="flex size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground shadow-sm">
          <Server className="size-4" aria-hidden />
        </span>
        <span className="text-lg font-bold tracking-tight">easy-infra</span>
      </div>
      <nav className="flex-1 space-y-1 p-4">
        <NavLink item={navItems[0]} route={route} />
        <ProfileNav />
        {navItems.slice(1).map((item) => (
          <NavLink key={item.label} item={item} route={route} />
        ))}
      </nav>
      <div className="border-t p-4">
        <ThemeToggle />
      </div>
    </aside>
  );
}
