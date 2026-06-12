import type { ReactNode } from "react";

import { cn } from "@/lib/utils";

import { Sidebar } from "./Sidebar";

interface AdminLayoutProps {
  children: ReactNode;
  /** Let the content span the full width beside the sidebar (no centered cap). */
  fullWidth?: boolean;
}

/**
 * Admin shell: a fixed sidebar alongside a scrollable main content area.
 * Owns the page chrome so feature screens only render their own content.
 * Most screens are capped and centered; pass `fullWidth` for screens that
 * should use all the remaining width.
 */
export function AdminLayout({ children, fullWidth = false }: AdminLayoutProps) {
  return (
    <div className="flex min-h-screen bg-background">
      <Sidebar />
      <main className="flex-1 overflow-y-auto">
        <div
          className={cn(
            "px-6 py-10",
            fullWidth ? "w-full" : "mx-auto max-w-5xl",
          )}
        >
          {children}
        </div>
      </main>
    </div>
  );
}
