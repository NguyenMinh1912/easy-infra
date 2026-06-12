import {
  Boxes,
  Database,
  LayoutDashboard,
  Layers,
  type LucideIcon,
} from "lucide-react";

import { cn } from "@/lib/utils";

/** The feature screens the sidebar can navigate between. */
export type View = "dashboard" | "profiles";

interface NavItem {
  label: string;
  icon: LucideIcon;
  /** The screen this item opens; items without one are not navigable yet. */
  view?: View;
}

/**
 * Navigation entries for the admin sidebar. Mirrors the tool's domain
 * (dashboard, profiles, services, backup). Dashboard and Profiles are wired up;
 * the rest are placeholders for future screens.
 */
const navItems: NavItem[] = [
  { label: "Dashboard", icon: LayoutDashboard, view: "dashboard" },
  { label: "Profiles", icon: Layers, view: "profiles" },
  { label: "Services", icon: Boxes },
  { label: "Backup", icon: Database },
];

interface SidebarProps {
  current: View;
  onNavigate: (view: View) => void;
}

/**
 * Admin sidebar: brand block plus primary navigation. Highlights the active
 * screen and reports navigation intent upward; the app shell owns which screen
 * is shown.
 */
export function Sidebar({ current, onNavigate }: SidebarProps) {
  return (
    <aside className="flex w-60 shrink-0 flex-col border-r bg-card">
      <div className="flex h-16 items-center border-b px-6">
        <span className="text-lg font-bold tracking-tight">easy-infra</span>
      </div>
      <nav className="flex-1 space-y-1 p-4">
        {navItems.map(({ label, icon: Icon, view }) => {
          const active = view === current;
          return (
            <button
              key={label}
              type="button"
              disabled={!view}
              aria-current={active ? "page" : undefined}
              onClick={view ? () => onNavigate(view) : undefined}
              className={cn(
                "flex w-full items-center gap-3 rounded-md px-3 py-2 text-left text-sm font-medium transition-colors",
                active
                  ? "bg-accent text-accent-foreground"
                  : "text-muted-foreground hover:bg-accent hover:text-accent-foreground",
                !view && "cursor-not-allowed opacity-50 hover:bg-transparent",
              )}
            >
              <Icon className="size-4 shrink-0" aria-hidden />
              {label}
            </button>
          );
        })}
      </nav>
    </aside>
  );
}
