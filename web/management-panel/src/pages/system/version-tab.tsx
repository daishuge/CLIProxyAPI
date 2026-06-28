import { GitBranch, RefreshCw } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useSystemLatestVersionQuery } from "@/lib/api/system";
import { APP_VERSION } from "@/lib/version";
import {
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  EmptyState,
  Skeleton,
} from "@/components/ui";

/** Version comparison card: current build vs latest upstream release. */
export function VersionTab() {
  const { t } = useTranslation();
  const query = useSystemLatestVersionQuery();

  const latest = query.data?.["latest-version"] ?? null;
  const updateAvailable = latest !== null && latest !== APP_VERSION;

  if (query.isLoading) {
    return <Skeleton className="h-40 w-full max-w-md" />;
  }

  if (query.isError) {
    return (
      <EmptyState
        tone="danger"
        title={t("table.error_title")}
        description={t("table.error_description")}
        action={
          <Button variant="outline" size="sm" onClick={() => void query.refetch()}>
            <RefreshCw className="size-4" />
            {t("common.retry")}
          </Button>
        }
      />
    );
  }

  return (
    <Card className="max-w-md">
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <GitBranch className="size-5 text-muted-foreground" />
          {t("system.version_title")}
        </CardTitle>
        <CardDescription>{t("system.version_desc")}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-3">
          <div className="flex items-center justify-between">
            <span className="text-sm text-muted-foreground">{t("system.current_version")}</span>
            <Badge variant="outline" className="font-mono">
              {APP_VERSION}
            </Badge>
          </div>
          {latest ? (
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted-foreground">{t("system.latest_version")}</span>
              <Badge variant={updateAvailable ? "default" : "outline"} className="font-mono">
                {latest}
              </Badge>
            </div>
          ) : null}
        </div>

        {updateAvailable ? (
          <Badge variant="default">{t("header.update_available")}</Badge>
        ) : latest ? (
          <Badge variant="outline">{t("header.up_to_date")}</Badge>
        ) : null}
      </CardContent>
    </Card>
  );
}
