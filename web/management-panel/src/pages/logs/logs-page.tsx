import { useTranslation } from "react-i18next";
import { PageHeader } from "@/components/layout/page-header";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui";
import { useHashTab } from "@/lib/use-hash-tab";
import { RequestLogsTab } from "./request-logs-tab";
import { ErrorLogsTab } from "./error-logs-tab";
import { ConversationLogsTab } from "./conversation-logs-tab";
import { LogStorageTab } from "./log-storage-tab";

const TABS = ["request", "errors", "conversations", "storage"] as const;

/**
 * Logs area. Hosts request logs, error logs, conversation logs (PPAP private)
 * and log storage management behind a tab strip.
 */
export function LogsPage() {
  const { t } = useTranslation();
  const [tab, setTab] = useHashTab(TABS, "request");

  return (
    <div className="space-y-6">
      <PageHeader title={t("nav.logs")} description={t("logs.subtitle")} />

      <Tabs value={tab} onValueChange={(value) => setTab(value as (typeof TABS)[number])}>
        <TabsList>
          <TabsTrigger value="request">{t("logs.tab_request")}</TabsTrigger>
          <TabsTrigger value="errors">{t("logs.tab_errors")}</TabsTrigger>
          <TabsTrigger value="conversations">{t("logs.tab_conversations")}</TabsTrigger>
          <TabsTrigger value="storage">{t("logs.tab_storage")}</TabsTrigger>
        </TabsList>
        <TabsContent value="request">
          <RequestLogsTab />
        </TabsContent>
        <TabsContent value="errors">
          <ErrorLogsTab />
        </TabsContent>
        <TabsContent value="conversations">
          <ConversationLogsTab />
        </TabsContent>
        <TabsContent value="storage">
          <LogStorageTab />
        </TabsContent>
      </Tabs>
    </div>
  );
}
