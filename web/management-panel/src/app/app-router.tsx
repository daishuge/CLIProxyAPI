import { type ReactNode } from "react";
import { useRouter } from "./router";
import { type RouteId } from "./routes";
import { OverviewPage } from "@/pages/overview-page";
import { ProvidersPage } from "@/pages/providers/providers-page";
import { RoutingPage } from "@/pages/routing/routing-page";
import { TrafficPage } from "@/pages/traffic/traffic-page";
import { LogsPage } from "@/pages/logs/logs-page";
import { PluginsPage } from "@/pages/plugins/plugins-page";
import { ConfigPage } from "@/pages/config/config-page";
import { SystemPage } from "@/pages/system/system-page";

/** Renders the active route's page component. */
export function AppRouter() {
  const { route } = useRouter();

  const pages: Record<RouteId, ReactNode> = {
    overview: <OverviewPage />,
    providers: <ProvidersPage />,
    routing: <RoutingPage />,
    plugins: <PluginsPage />,
    traffic: <TrafficPage />,
    logs: <LogsPage />,
    config: <ConfigPage />,
    system: <SystemPage />,
  };

  return pages[route.id];
}
