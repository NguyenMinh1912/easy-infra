import { AdminLayout } from "@/components/layout/AdminLayout";
import { PageHeader } from "@/components/layout/PageHeader";
import { DashboardPage } from "@/features/dashboard";

/**
 * App shell. Owns the page chrome (admin layout, header) and mounts feature
 * screens. Keep this thin: routing and feature wiring live here, business
 * logic does not.
 */
export default function App() {
  return (
    <AdminLayout>
      <PageHeader
        title="Dashboard"
        subtitle="Local/dev infrastructure overview"
      />
      <DashboardPage />
    </AdminLayout>
  );
}
