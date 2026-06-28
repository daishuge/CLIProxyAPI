import { RefreshCw, Shield } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useAuthStatusQuery } from "@/lib/api/system";
import {
  Button,
  Card,
  CardDescription,
  CardHeader,
  CardTitle,
  EmptyState,
  Skeleton,
} from "@/components/ui";

/** Auth status dashboard — active providers, total auth files, etc. */
export function AuthStatusTab() {
  const { t } = useTranslation();
  const query = useAuthStatusQuery();

  if (query.isLoading) {
    return (
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} className="h-28 w-full" />
        ))}
      </div>
    );
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

  const data = query.data;
  const activeProviders = data?.["active-providers"] ?? 0;
  const totalAuthFiles = data?.["total-auth-files"] ?? 0;

  // Collect any extra numeric fields the backend may include.
  const extraFields = Object.entries(data ?? {}).filter(
    ([key, value]) =>
      key !== "active-providers" &&
      key !== "total-auth-files" &&
      typeof value === "number",
  );

  return (
    <div className="space-y-4">
      <div className="flex justify-end">
        <Button
          variant="outline"
          size="sm"
          onClick={() => void query.refetch()}
          disabled={query.isFetching}
        >
          <RefreshCw className="size-4" />
          {t("common.refresh")}
        </Button>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {/* Active Providers */}
        <Card>
          <CardHeader className="pb-2">
            <CardDescription>{t("system.auth_active_providers")}</CardDescription>
            <CardTitle className="flex items-center gap-2 text-2xl">
              <Shield className="size-5 text-muted-foreground" />
              {activeProviders}
            </CardTitle>
          </CardHeader>
        </Card>

        {/* Total Auth Files */}
        <Card>
          <CardHeader className="pb-2">
            <CardDescription>{t("system.auth_total_files")}</CardDescription>
            <CardTitle className="text-2xl">{totalAuthFiles}</CardTitle>
          </CardHeader>
        </Card>

        {/* Extra numeric fields from the backend */}
        {extraFields.map(([key, value]) => (
          <Card key={key}>
            <CardHeader className="pb-2">
              <CardDescription>{key.replace(/-/g, " ")}</CardDescription>
              <CardTitle className="text-2xl">{String(value)}</CardTitle>
            </CardHeader>
          </Card>
        ))}
      </div>
    </div>
  );
}
