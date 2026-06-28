import { useTranslation } from "react-i18next";
import { PageHeader } from "@/components/layout/page-header";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui";
import { useHashTab } from "@/lib/use-hash-tab";
import { AuthFilesTab } from "./auth-files-tab";
import { OAuthLoginsTab } from "./oauth-logins-tab";
import { ProviderKeysTab } from "./provider-keys-tab";

const TABS = ["auth-files", "oauth", "api-keys"] as const;

/**
 * Providers & Auth area. Hosts auth-file management, OAuth login flows and
 * provider API-key administration behind a tab strip. The active tab is
 * reflected in the hash query so deep links survive reloads.
 */
export function ProvidersPage() {
  const { t } = useTranslation();
  const [tab, setTab] = useHashTab(TABS, "auth-files");

  return (
    <div className="space-y-6">
      <PageHeader title={t("nav.providers")} description={t("providers.subtitle")} />

      <Tabs value={tab} onValueChange={(value) => setTab(value as (typeof TABS)[number])}>
        <TabsList>
          <TabsTrigger value="auth-files">{t("providers.tab_auth_files")}</TabsTrigger>
          <TabsTrigger value="oauth">{t("providers.tab_oauth")}</TabsTrigger>
          <TabsTrigger value="api-keys">{t("providers.tab_api_keys")}</TabsTrigger>
        </TabsList>
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
