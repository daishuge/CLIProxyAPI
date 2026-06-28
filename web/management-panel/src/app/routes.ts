import {
  LayoutDashboard,
  Plug,
  Route as RouteIcon,
  Puzzle,
  Activity,
  ScrollText,
  Settings2,
  Server,
  type LucideIcon,
} from "lucide-react";

/** Stable route identifiers used by the hash router and navigation. */
export type RouteId =
  | "overview"
  | "providers"
  | "routing"
  | "plugins"
  | "traffic"
  | "logs"
  | "config"
  | "system";

export interface RouteDefinition {
  id: RouteId;
  /** Hash path without the leading "#". */
  path: string;
  /** i18n key under `nav.*` for the label. */
  labelKey: string;
  icon: LucideIcon;
  /** Sidebar grouping. */
  group: "main" | "platform";
}

export const ROUTES: RouteDefinition[] = [
  { id: "overview", path: "/overview", labelKey: "nav.overview", icon: LayoutDashboard, group: "main" },
  { id: "providers", path: "/providers", labelKey: "nav.providers", icon: Plug, group: "main" },
  { id: "routing", path: "/routing", labelKey: "nav.routing", icon: RouteIcon, group: "main" },
  { id: "plugins", path: "/plugins", labelKey: "nav.plugins", icon: Puzzle, group: "main" },
  { id: "traffic", path: "/traffic", labelKey: "nav.traffic", icon: Activity, group: "main" },
  { id: "logs", path: "/logs", labelKey: "nav.logs", icon: ScrollText, group: "platform" },
  { id: "config", path: "/config", labelKey: "nav.config", icon: Settings2, group: "platform" },
  { id: "system", path: "/system", labelKey: "nav.system", icon: Server, group: "platform" },
];

export const DEFAULT_ROUTE: RouteDefinition = ROUTES[0]!;

/** Resolve a route definition from a hash path; falls back to the default. */
export function routeFromPath(path: string): RouteDefinition {
  const normalized = path.split("?")[0] ?? "";
  return ROUTES.find((route) => route.path === normalized) ?? DEFAULT_ROUTE;
}
