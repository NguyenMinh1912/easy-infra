import type { ReactNode } from "react";

import { Sidebar, type View } from "./Sidebar";

interface AdminLayoutProps {
  children: ReactNode;
  current: View;
  onNavigate: (view: View) => void;
}

/**
 * Admin shell: a fixed sidebar alongside a scrollable main content area.
 * Owns the page chrome so feature screens only render their own content; the
 * active screen is driven by the app shell through {@link Sidebar}.
 */
export function AdminLayout({ children, current, onNavigate }: AdminLayoutProps) {
  return (
    <div className="flex min-h-screen bg-background">
      <Sidebar current={current} onNavigate={onNavigate} />
      <main className="flex-1 overflow-y-auto">
        <div className="mx-auto max-w-5xl px-6 py-10">{children}</div>
      </main>
    </div>
  );
}
