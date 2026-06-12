import { PageHeader } from "@/components/layout/PageHeader";
import { DashboardPage } from "@/features/dashboard";

/**
 * App shell. Owns the page chrome (layout, header) and mounts feature screens.
 * Keep this thin: routing and feature wiring live here, business logic does
 * not.
 */
export default function App() {
  return (
    <div className="min-h-screen bg-background">
      <main className="mx-auto max-w-5xl px-6 py-10">
        <PageHeader
          title="easy-infra"
          subtitle="Local/dev infrastructure dashboard"
        />
        <DashboardPage />
      </main>
    </div>
  );
}
