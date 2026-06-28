import { useTranslation } from "react-i18next";
import { PageHeader } from "@/components/layout/page-header";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui";
import { useHashTab } from "@/lib/use-hash-tab";
import { CustomUpstreamsTab } from "./custom-upstreams-tab";
import { ModelsTab } from "./models-tab";
import { ModelAliasesTab } from "./model-aliases-tab";
import { RoutingStrategyTab } from "./routing-strategy-tab";

const TABS = ["upstreams", "models", "aliases", "strategy"] as const;

/**
 * Routing & Models area. Hosts the custom-upstreams (headline), model catalog,
 * model aliases, and routing-strategy sub-features behind a tab strip. The
 * active tab is reflected in the hash query so deep links survive reloads.
 */
export function RoutingPage() {
  const { t } = useTranslation();
  const [tab, setTab] = useHashTab(TABS, "upstreams");

  return (
    <div className="space-y-6">
      <PageHeader title={t("nav.routing")} description={t("routing.subtitle")} />

      <Tabs value={tab} onValueChange={(value) => setTab(value as (typeof TABS)[number])}>
        <TabsList>
          <TabsTrigger value="upstreams">{t("routing.tab_upstreams")}</TabsTrigger>
          <TabsTrigger value="models">{t("routing.tab_models")}</TabsTrigger>
          <TabsTrigger value="aliases">{t("routing.tab_aliases")}</TabsTrigger>
          <TabsTrigger value="strategy">{t("routing.tab_strategy")}</TabsTrigger>
        </TabsList>
        <TabsContent value="upstreams">
          <CustomUpstreamsTab />
        </TabsContent>
        <TabsContent value="models">
          <ModelsTab />
        </TabsContent>
        <TabsContent value="aliases">
          <ModelAliasesTab />
        </TabsContent>
        <TabsContent value="strategy">
          <RoutingStrategyTab />
        </TabsContent>
      </Tabs>
    </div>
  );
}
