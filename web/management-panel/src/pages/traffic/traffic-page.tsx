import { useTranslation } from "react-i18next";
import { PageHeader } from "@/components/layout/page-header";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui";
import { useHashTab } from "@/lib/use-hash-tab";
import { UsageTab } from "./usage-tab";
import { ApiKeyUsageTab } from "./api-key-usage-tab";

const TABS = ["usage", "api-keys"] as const;

/**
 * Traffic & Usage area. Hosts aggregate usage statistics and per-API-key
 * consumption data behind a tab strip. The active tab is reflected in the
 * hash query so deep links survive reloads.
 */
export function TrafficPage() {
  const { t } = useTranslation();
  const [tab, setTab] = useHashTab(TABS, "usage");

  return (
    <div className="space-y-6">
      <PageHeader title={t("nav.traffic")} description={t("traffic.subtitle")} />

      <Tabs value={tab} onValueChange={(value) => setTab(value as (typeof TABS)[number])}>
        <TabsList>
          <TabsTrigger value="usage">{t("traffic.tab_usage")}</TabsTrigger>
          <TabsTrigger value="api-keys">{t("traffic.tab_api_keys")}</TabsTrigger>
        </TabsList>
        <TabsContent value="usage">
          <UsageTab />
        </TabsContent>
        <TabsContent value="api-keys">
          <ApiKeyUsageTab />
        </TabsContent>
      </Tabs>
    </div>
  );
}
