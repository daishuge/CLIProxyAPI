import { useTranslation } from "react-i18next";
import { PageHeader } from "@/components/layout/page-header";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui";
import { useHashTab } from "@/lib/use-hash-tab";
import { CustomUpstreamsTab } from "./custom-upstreams-tab";
import { AuthFilesTab } from "./auth-files-tab";
import { OAuthLoginsTab } from "./oauth-logins-tab";
import { ProviderKeysTab } from "./provider-keys-tab";

const TABS = ["upstreams", "auth-files", "oauth", "api-keys"] as const;

export function ProvidersPage() {
  const { t } = useTranslation();
  const [tab, setTab] = useHashTab(TABS, "upstreams");

  return (
    <div className="space-y-6">
      <PageHeader title={t("nav.providers")} description={t("providers.subtitle")} />

      <Tabs value={tab} onValueChange={(value) => setTab(value as (typeof TABS)[number])}>
        <TabsList>
          <TabsTrigger value="upstreams">{t("routing.tab_upstreams")}</TabsTrigger>
          <TabsTrigger value="auth-files">{t("providers.tab_auth_files")}</TabsTrigger>
          <TabsTrigger value="oauth">{t("providers.tab_oauth")}</TabsTrigger>
          <TabsTrigger value="api-keys">{t("providers.tab_api_keys")}</TabsTrigger>
        </TabsList>
        <TabsContent value="upstreams">
          <CustomUpstreamsTab />
        </TabsContent>
        <TabsContent value="auth-files">
          <AuthFilesTab />
        </TabsContent>
        <TabsContent value="oauth">
          <OAuthLoginsTab />
        </TabsContent>
        <TabsContent value="api-keys">
          <ProviderKeysTab />
        </TabsContent>
      </Tabs>
    </div>
  );
}
