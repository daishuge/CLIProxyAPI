import { useTranslation } from "react-i18next";
import { PageHeader } from "@/components/layout/page-header";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui";
import { useHashTab } from "@/lib/use-hash-tab";
import { ModelsTab } from "./models-tab";
import { ModelAliasesTab } from "./model-aliases-tab";
import { RoutingStrategyTab } from "./routing-strategy-tab";

const TABS = ["models", "aliases", "strategy"] as const;

export function RoutingPage() {
  const { t } = useTranslation();
  const [tab, setTab] = useHashTab(TABS, "models");

  return (
    <div className="space-y-6">
      <PageHeader title={t("nav.routing")} description={t("routing.subtitle")} />

      <Tabs value={tab} onValueChange={(value) => setTab(value as (typeof TABS)[number])}>
        <TabsList>
          <TabsTrigger value="models">{t("routing.tab_models")}</TabsTrigger>
          <TabsTrigger value="aliases">{t("routing.tab_aliases")}</TabsTrigger>
          <TabsTrigger value="strategy">{t("routing.tab_strategy")}</TabsTrigger>
        </TabsList>
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
