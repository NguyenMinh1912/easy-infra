import { AdminLayout } from "@/components/layout/AdminLayout";
import { PageHeader } from "@/components/layout/PageHeader";
import { DashboardPage } from "@/features/dashboard";
import { ServicesPage } from "@/features/services";
import { useHashRoute } from "@/hooks/useHashRoute";

/**
 * App shell. Owns the page chrome (admin layout, header) and selects the active
 * screen from the hash route. Keep this thin: routing and feature wiring live
 * here, business logic does not.
 */
export default function App() {
  const route = useHashRoute();

  return (
    <AdminLayout>
      {route.startsWith("/services") ? (
        <>
          <PageHeader
            title="Services"
            subtitle="Manage the services your project defines"
          />
          <ServicesPage />
        </>
      ) : (
        <>
          <PageHeader
            title="Dashboard"
            subtitle="Local/dev infrastructure overview"
          />
          <DashboardPage />
        </>
      )}
    </AdminLayout>
  );
}
