import type { ReactNode } from "react";

import { AdminLayout } from "@/components/layout/AdminLayout";
import { PageHeader } from "@/components/layout/PageHeader";
import { Toaster } from "@/components/ui/sonner";
import { BackupsPage } from "@/features/backups";
import { ProfilesPage } from "@/features/profiles";
import { metaFor, ServiceDetailPage } from "@/features/services";
import { useHashRoute } from "@/hooks/useHashRoute";

interface Screen {
  title: string;
  subtitle: string;
  content: ReactNode;
  /** Render the screen at full remaining width instead of the centered cap. */
  fullWidth?: boolean;
  /** Hide the page header to reclaim vertical space (screen renders its own). */
  hideHeader?: boolean;
}

/** Map the current hash route onto the screen to render. */
function screenForRoute(route: string): Screen {
  if (route.startsWith("/backups")) {
    return {
      title: "Backups",
      subtitle: "Browse, inspect, and remove backup sessions",
      content: <BackupsPage />,
    };
  }
  const profileService = route.match(/^\/profiles\/(.+)\/services\/(.+)$/);
  if (profileService) {
    const profile = decodeURIComponent(profileService[1]);
    const name = decodeURIComponent(profileService[2]);
    const meta = metaFor(name);
    return {
      title: meta.label,
      subtitle: meta.blurb,
      content: <ServiceDetailPage name={name} profile={profile} />,
      fullWidth: true,
      hideHeader: true,
    };
  }
  return {
    title: "Profiles",
    subtitle: "Create, switch, and remove environment profiles",
    content: <ProfilesPage />,
  };
}

/**
 * App shell. Owns the page chrome (admin layout, header) and selects the active
 * screen from the hash route. Keep this thin: routing and feature wiring live
 * here, business logic does not.
 */
export default function App() {
  const route = useHashRoute();
  const { title, subtitle, content, fullWidth, hideHeader } =
    screenForRoute(route);

  return (
    <AdminLayout fullWidth={fullWidth}>
      {!hideHeader && <PageHeader title={title} subtitle={subtitle} />}
      {content}
      <Toaster position="bottom-right" richColors closeButton />
    </AdminLayout>
  );
}
