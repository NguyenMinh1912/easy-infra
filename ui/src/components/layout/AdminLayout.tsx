import type { ReactNode } from "react";

import { Sidebar } from "./Sidebar";

interface AdminLayoutProps {
  children: ReactNode;
}

/**
 * Admin shell: a fixed sidebar alongside a scrollable main content area.
 * Owns the page chrome so feature screens only render their own content.
 */
export function AdminLayout({ children }: AdminLayoutProps) {
  return (
    <div className="flex min-h-screen bg-background">
      <Sidebar />
      <main className="flex-1 overflow-y-auto">
        <div className="mx-auto max-w-5xl px-6 py-10">{children}</div>
      </main>
    </div>
  );
}
