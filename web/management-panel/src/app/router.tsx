import * as React from "react";
import { DEFAULT_ROUTE, routeFromPath, type RouteDefinition } from "./routes";

/**
 * Minimal hash-based router. The panel ships as a single `management.html` with
 * no server-side fallback, so hash routing keeps deep links working without any
 * backend route rewrites.
 */

interface RouterContextValue {
  route: RouteDefinition;
  navigate: (path: string) => void;
}

const RouterContext = React.createContext<RouterContextValue | null>(null);

function currentHashPath(): string {
  const hash = window.location.hash.replace(/^#/, "");
  return hash || DEFAULT_ROUTE.path;
}

export function HashRouterProvider({ children }: { children: React.ReactNode }) {
  const [path, setPath] = React.useState<string>(() => currentHashPath());

  React.useEffect(() => {
    const onHashChange = () => setPath(currentHashPath());
    window.addEventListener("hashchange", onHashChange);
    // Normalize an empty hash to the default route on first paint.
    if (!window.location.hash) {
      window.location.hash = DEFAULT_ROUTE.path;
    }
    return () => window.removeEventListener("hashchange", onHashChange);
  }, []);

  const navigate = React.useCallback((next: string) => {
    const target = next.startsWith("/") ? next : `/${next}`;
    if (window.location.hash.replace(/^#/, "") !== target) {
      window.location.hash = target;
    }
  }, []);

  const value = React.useMemo<RouterContextValue>(
    () => ({ route: routeFromPath(path), navigate }),
    [path, navigate],
  );

  return <RouterContext.Provider value={value}>{children}</RouterContext.Provider>;
}

export function useRouter(): RouterContextValue {
  const ctx = React.useContext(RouterContext);
  if (!ctx) throw new Error("useRouter must be used within HashRouterProvider");
  return ctx;
}
