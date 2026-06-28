import { useTranslation } from "react-i18next";
import { ROUTES, type RouteDefinition } from "@/app/routes";
import { useRouter } from "@/app/router";
import { LogoLockup } from "@/components/brand/logo";
import { cn } from "@/lib/utils";

function NavItem({ route, active }: { route: RouteDefinition; active: boolean }) {
  const { t } = useTranslation();
  const { navigate } = useRouter();
  const Icon = route.icon;
  return (
    <button
      type="button"
      onClick={() => navigate(route.path)}
      aria-current={active ? "page" : undefined}
      className={cn(
        "group relative flex w-full items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40",
        active
          ? "bg-sidebar-accent text-foreground"
          : "text-sidebar-foreground hover:bg-sidebar-accent/60 hover:text-foreground",
      )}
    >
      {active ? (
        <span className="absolute left-0 top-1/2 h-5 w-0.5 -translate-y-1/2 rounded-full bg-primary" />
      ) : null}
      <Icon className={cn("size-4 shrink-0", active ? "text-primary" : "text-muted-foreground")} />
      <span className="truncate">{t(route.labelKey)}</span>
    </button>
  );
}

/** Left navigation rail with grouped sections and the brand lockup. */
export function Sidebar() {
  const { t } = useTranslation();
  const { route } = useRouter();

  const groups: { key: string; labelKey: string; items: RouteDefinition[] }[] = [
    { key: "main", labelKey: "nav.section_main", items: ROUTES.filter((r) => r.group === "main") },
    {
      key: "platform",
      labelKey: "nav.section_platform",
      items: ROUTES.filter((r) => r.group === "platform"),
    },
  ];

  return (
    <aside className="flex h-full w-60 shrink-0 flex-col border-r border-sidebar-border bg-sidebar">
      <div className="flex h-14 items-center border-b border-sidebar-border px-4">
        <LogoLockup showTagline />
      </div>
      <nav className="flex-1 space-y-6 overflow-y-auto px-3 py-4">
        {groups.map((group) => (
          <div key={group.key} className="space-y-1">
            <p className="px-3 pb-1 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground/70">
              {t(group.labelKey)}
            </p>
            {group.items.map((item) => (
              <NavItem key={item.id} route={item} active={route.id === item.id} />
            ))}
          </div>
        ))}
      </nav>
    </aside>
  );
}
