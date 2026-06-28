import * as React from "react";
import { type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { AlertTriangle, RefreshCw } from "lucide-react";
import { useRouter } from "./router";
import { type RouteId } from "./routes";
import { Button } from "@/components/ui";
import { OverviewPage } from "@/pages/overview-page";
import { ProvidersPage } from "@/pages/providers/providers-page";
import { RoutingPage } from "@/pages/routing/routing-page";
import { TrafficPage } from "@/pages/traffic/traffic-page";
import { LogsPage } from "@/pages/logs/logs-page";
import { PluginsPage } from "@/pages/plugins/plugins-page";
import { ConfigPage } from "@/pages/config/config-page";
import { SystemPage } from "@/pages/system/system-page";

/**
 * Page-level error boundary that auto-resets on navigation. A crash in one
 * page no longer takes down the sidebar, topbar, or other tabs.
 */
class PageErrorBoundary extends React.Component<
  { routeId: string; children: ReactNode; fallback: ReactNode },
  { hasError: boolean }
> {
  constructor(props: { routeId: string; children: ReactNode; fallback: ReactNode }) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(): { hasError: boolean } {
    return { hasError: true };
  }

  componentDidCatch(error: Error, info: React.ErrorInfo): void {
    console.error("Page render error:", error, info.componentStack);
  }

  componentDidUpdate(prevProps: { routeId: string }): void {
    // Auto-reset when the user navigates away from the crashed page.
    if (prevProps.routeId !== this.props.routeId && this.state.hasError) {
      this.setState({ hasError: false });
    }
  }

  override render(): ReactNode {
    return this.state.hasError ? this.props.fallback : this.props.children;
  }
}

function PageCrashFallback() {
  const { t } = useTranslation();
  return (
    <div className="flex flex-col items-center justify-center gap-4 py-24 text-center">
      <span className="flex size-10 items-center justify-center rounded-full bg-warning/10 text-warning">
        <AlertTriangle className="size-5" />
      </span>
      <div className="space-y-1">
        <h2 className="text-base font-semibold text-foreground">{t("error.page_crash_title")}</h2>
        <p className="max-w-sm text-sm text-muted-foreground">{t("error.page_crash_desc")}</p>
      </div>
      <Button variant="outline" size="sm" onClick={() => window.location.reload()}>
        <RefreshCw className="size-4" />
        {t("error.reload")}
      </Button>
    </div>
  );
}

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

  return (
    <PageErrorBoundary routeId={route.id} fallback={<PageCrashFallback />}>
      {pages[route.id]}
    </PageErrorBoundary>
  );
}
