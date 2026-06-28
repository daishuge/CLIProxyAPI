import { useTranslation } from "react-i18next";
import { PageHeader } from "@/components/layout/page-header";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui";
import { useHashTab } from "@/lib/use-hash-tab";
import { InstalledPluginsTab } from "./installed-plugins-tab";
import { PluginStoreTab } from "./plugin-store-tab";

const TABS = ["installed", "store"] as const;

/**
 * Plugins area. Two tabs: installed plugin management and the plugin store
 * for discovering and installing new plugins.
 */
export function PluginsPage() {
  const { t } = useTranslation();
  const [tab, setTab] = useHashTab(TABS, "installed");

  return (
    <div className="space-y-6">
      <PageHeader title={t("nav.plugins")} description={t("plugins.subtitle")} />

      <Tabs value={tab} onValueChange={(value) => setTab(value as (typeof TABS)[number])}>
        <TabsList>
          <TabsTrigger value="installed">{t("plugins.tab_installed")}</TabsTrigger>
          <TabsTrigger value="store">{t("plugins.tab_store")}</TabsTrigger>
        </TabsList>
        <TabsContent value="installed">
          <InstalledPluginsTab />
        </TabsContent>
        <TabsContent value="store">
          <PluginStoreTab />
        </TabsContent>
      </Tabs>
    </div>
  );
}
