import { useTranslation } from "react-i18next";
import { PageHeader } from "@/components/layout/page-header";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui";
import { useHashTab } from "@/lib/use-hash-tab";
import { VisualConfigTab } from "./visual-config-tab";
import { YamlEditorTab } from "./yaml-editor-tab";
import { DownstreamKeysTab } from "./downstream-keys-tab";

const TABS = ["visual", "yaml", "api-keys"] as const;

/**
 * Config area. Three tabs: visual config editor, raw YAML editor, and
 * downstream API key management.
 */
export function ConfigPage() {
  const { t } = useTranslation();
  const [tab, setTab] = useHashTab(TABS, "visual");

  return (
    <div className="space-y-6">
      <PageHeader title={t("nav.config")} description={t("config.subtitle")} />

      <Tabs value={tab} onValueChange={(value) => setTab(value as (typeof TABS)[number])}>
        <TabsList>
          <TabsTrigger value="visual">{t("config.tab_visual")}</TabsTrigger>
          <TabsTrigger value="yaml">{t("config.tab_yaml")}</TabsTrigger>
          <TabsTrigger value="api-keys">{t("config.tab_api_keys")}</TabsTrigger>
        </TabsList>
        <TabsContent value="visual">
          <VisualConfigTab />
        </TabsContent>
        <TabsContent value="yaml">
          <YamlEditorTab />
        </TabsContent>
        <TabsContent value="api-keys">
          <DownstreamKeysTab />
        </TabsContent>
      </Tabs>
    </div>
  );
}
