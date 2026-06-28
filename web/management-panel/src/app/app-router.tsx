import { type ReactNode } from "react";
import { useRouter } from "./router";
import { type RouteId } from "./routes";
import { OverviewPage } from "@/pages/overview-page";
import { PlaceholderPage } from "@/pages/placeholder-page";

/** Renders the active route's page component. */
export function AppRouter() {
  const { route } = useRouter();

  const pages: Record<RouteId, ReactNode> = {
    overview: <OverviewPage />,
    providers: <PlaceholderPage titleKey="nav.providers" />,
    routing: <PlaceholderPage titleKey="nav.routing" />,
    plugins: <PlaceholderPage titleKey="nav.plugins" />,
    traffic: <PlaceholderPage titleKey="nav.traffic" />,
    logs: <PlaceholderPage titleKey="nav.logs" />,
    config: <PlaceholderPage titleKey="nav.config" />,
    system: <PlaceholderPage titleKey="nav.system" />,
  };

  return pages[route.id];
}
