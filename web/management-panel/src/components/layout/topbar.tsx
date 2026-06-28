import { useTranslation } from "react-i18next";
import { useRouter } from "@/app/router";
import { HealthPill } from "./health-pill";
import { VersionPill } from "./version-pill";
import { ThemeToggle } from "./theme-toggle";
import { LanguageMenu } from "./language-menu";
import { AccountMenu } from "./account-menu";

/** Top bar: current section title, backend health/version, and global controls. */
export function Topbar() {
  const { t } = useTranslation();
  const { route } = useRouter();

  return (
    <header className="flex h-14 shrink-0 items-center justify-between gap-4 border-b border-border bg-surface/80 px-5 backdrop-blur">
      <div className="flex min-w-0 items-center gap-3">
        <h1 className="truncate text-sm font-semibold text-foreground">{t(route.labelKey)}</h1>
      </div>
      <div className="flex items-center gap-2">
        <div className="hidden items-center gap-2 sm:flex">
          <HealthPill />
          <VersionPill />
        </div>
        <div className="mx-1 hidden h-5 w-px bg-border sm:block" />
        <LanguageMenu />
        <ThemeToggle />
        <AccountMenu />
      </div>
    </header>
  );
}
