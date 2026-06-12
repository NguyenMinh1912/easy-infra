import { useState } from "react";

import { AdminLayout } from "@/components/layout/AdminLayout";
import { PageHeader } from "@/components/layout/PageHeader";
import type { View } from "@/components/layout/Sidebar";
import { DashboardPage } from "@/features/dashboard";
import { ProfilesPage } from "@/features/profiles";

/**
 * App shell. Owns the page chrome (admin layout, header) and the active screen,
 * and mounts feature screens. Keep this thin: routing and feature wiring live
 * here, business logic does not.
 */
export default function App() {
  const [view, setView] = useState<View>("dashboard");

  return (
    <AdminLayout current={view} onNavigate={setView}>
      {view === "dashboard" ? (
        <>
          <PageHeader
            title="Dashboard"
            subtitle="Local/dev infrastructure overview"
          />
          <DashboardPage />
        </>
      ) : (
        <>
          <PageHeader
            title="Profiles"
            subtitle="Create, switch, and remove environment profiles"
          />
          <ProfilesPage />
        </>
      )}
    </AdminLayout>
  );
}
