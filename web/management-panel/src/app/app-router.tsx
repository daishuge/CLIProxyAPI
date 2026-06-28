import { type ReactNode } from "react";
import { useRouter } from "./router";
import { type RouteId } from "./routes";
import { OverviewPage } from "@/pages/overview-page";
import { PlaceholderPage } from "@/pages/placeholder-page";
import { ProvidersPage } from "@/pages/providers/providers-page";
import { RoutingPage } from "@/pages/routing/routing-page";

/** Renders the active route's page component. */
export function AppRouter() {
  const { route } = useRouter();

  const pages: Record<RouteId, ReactNode> = {
    overview: <OverviewPage />,
    providers: <ProvidersPage />,
    routing: <RoutingPage />,
    plugins: <PlaceholderPage titleKey="nav.plugins" />,
    traffic: <PlaceholderPage titleKey="nav.traffic" />,
    logs: <PlaceholderPage titleKey="nav.logs" />,
    config: <PlaceholderPage titleKey="nav.config" />,
    system: <PlaceholderPage titleKey="nav.system" />,
  };

  return pages[route.id];
}
