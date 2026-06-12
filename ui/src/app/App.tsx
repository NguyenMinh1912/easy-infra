import type { ReactNode } from "react";

import { AdminLayout } from "@/components/layout/AdminLayout";
import { PageHeader } from "@/components/layout/PageHeader";
import { Toaster } from "@/components/ui/sonner";
import { DashboardPage } from "@/features/dashboard";
import { ProfilesPage, ProfileSettingsPage } from "@/features/profiles";
import { metaFor, ServiceDetailPage, ServicesPage } from "@/features/services";
import { useHashRoute } from "@/hooks/useHashRoute";

interface Screen {
  title: string;
  subtitle: string;
  content: ReactNode;
}

/** Map the current hash route onto the screen to render. */
function screenForRoute(route: string): Screen {
  const serviceDetail = route.match(/^\/services\/(.+)$/);
  if (serviceDetail) {
    const name = decodeURIComponent(serviceDetail[1]);
    const meta = metaFor(name);
    return {
      title: meta.label,
      subtitle: meta.blurb,
      content: <ServiceDetailPage name={name} />,
    };
  }
  if (route.startsWith("/services")) {
    return {
      title: "Services",
      subtitle: "Manage the services your project defines",
      content: <ServicesPage />,
    };
  }
  const settings = route.match(/^\/profiles\/(.+)\/settings$/);
  if (settings) {
    const name = decodeURIComponent(settings[1]);
    return {
      title: "Profile settings",
      subtitle: `Configure the "${name}" profile`,
      content: <ProfileSettingsPage name={name} />,
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
