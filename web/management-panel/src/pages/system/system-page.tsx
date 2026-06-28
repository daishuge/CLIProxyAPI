import { useTranslation } from "react-i18next";
import { PageHeader } from "@/components/layout/page-header";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui";
import { useHashTab } from "@/lib/use-hash-tab";
import { HealthTab } from "./health-tab";
import { VersionTab } from "./version-tab";
import { AuthStatusTab } from "./auth-status-tab";

const TABS = ["health", "version", "auth-status"] as const;

/**
 * System area. Three tabs: health check with auto-polling, version comparison,
 * and auth status overview.
 */
export function SystemPage() {
  const { t } = useTranslation();
  const [tab, setTab] = useHashTab(TABS, "health");

  return (
    <div className="space-y-6">
      <PageHeader title={t("nav.system")} description={t("system.subtitle")} />

      <Tabs value={tab} onValueChange={(value) => setTab(value as (typeof TABS)[number])}>
        <TabsList>
          <TabsTrigger value="health">{t("system.tab_health")}</TabsTrigger>
          <TabsTrigger value="version">{t("system.tab_version")}</TabsTrigger>
          <TabsTrigger value="auth-status">{t("system.tab_auth_status")}</TabsTrigger>
        </TabsList>
        <TabsContent value="health">
          <HealthTab />
        </TabsContent>
        <TabsContent value="version">
          <VersionTab />
        </TabsContent>
        <TabsContent value="auth-status">
          <AuthStatusTab />
        </TabsContent>
      </Tabs>
    </div>
  );
}
