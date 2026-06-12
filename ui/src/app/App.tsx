import type { ReactNode } from "react";

import { AdminLayout } from "@/components/layout/AdminLayout";
import { PageHeader } from "@/components/layout/PageHeader";
import { Toaster } from "@/components/ui/sonner";
import { DashboardPage } from "@/features/dashboard";
import { ProfilesPage } from "@/features/profiles";
import { ServicesPage } from "@/features/services";
import { useHashRoute } from "@/hooks/useHashRoute";

interface Screen {
  title: string;
  subtitle: string;
  content: ReactNode;
}

/** Map the current hash route onto the screen to render. */
function screenForRoute(route: string): Screen {
  if (route.startsWith("/services")) {
    return {
      title: "Services",
      subtitle: "Manage the services your project defines",
      content: <ServicesPage />,
    };
  }
  if (route.startsWith("/profiles")) {
    return {
      title: "Profiles",
      subtitle: "Create, switch, and remove environment profiles",
      content: <ProfilesPage />,
    };
  }
  return {
    title: "Dashboard",
    subtitle: "Local/dev infrastructure overview",
    content: <DashboardPage />,
  };
}

/**
 * App shell. Owns the page chrome (admin layout, header) and selects the active
 * screen from the hash route. Keep this thin: routing and feature wiring live
 * here, business logic does not.
 */
export default function App() {
  const route = useHashRoute();
  const { title, subtitle, content } = screenForRoute(route);

  return (
    <AdminLayout>
      <PageHeader title={title} subtitle={subtitle} />
      {content}
      <Toaster position="bottom-right" richColors closeButton />
    </AdminLayout>
  );
}
