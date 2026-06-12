import {
  Boxes,
  Database,
  LayoutDashboard,
  Layers,
  type LucideIcon,
} from "lucide-react";

import { cn } from "@/lib/utils";

interface NavItem {
  label: string;
  icon: LucideIcon;
  /** Whether this item is the currently active screen. */
  active?: boolean;
}

/**
 * Navigation entries for the admin sidebar. Mirrors the tool's domain
 * (dashboard, profiles, services, backup). The app has no router yet, so
 * only the dashboard is active; the rest are placeholders for future screens.
 */
const navItems: NavItem[] = [
  { label: "Dashboard", icon: LayoutDashboard, active: true },
  { label: "Profiles", icon: Layers },
  { label: "Services", icon: Boxes },
  { label: "Backup", icon: Database },
];

/**
 * Admin sidebar: brand block plus primary navigation. Presentational only —
 * it renders the static nav and highlights the active item.
 */
export function Sidebar() {
  return (
    <aside className="flex w-60 shrink-0 flex-col border-r bg-card">
      <div className="flex h-16 items-center border-b px-6">
        <span className="text-lg font-bold tracking-tight">easy-infra</span>
      </div>
      <nav className="flex-1 space-y-1 p-4">
        {navItems.map(({ label, icon: Icon, active }) => (
          <a
            key={label}
            href="#"
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
        ))}
      </nav>
    </aside>
  );
}
