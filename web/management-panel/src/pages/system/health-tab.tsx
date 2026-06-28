import { Activity, RefreshCw } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useHealthQuery } from "@/lib/hooks";
import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  EmptyState,
  Skeleton,
} from "@/components/ui";
import { cn } from "@/lib/utils";

/** Health check card with auto-polling (30 s) and green/red indicator. */
export function HealthTab() {
  const { t } = useTranslation();
  const query = useHealthQuery();

  const healthy = query.data?.status === "ok" && !query.isError;
  const state: "healthy" | "offline" | "checking" = query.isLoading
    ? "checking"
    : healthy
      ? "healthy"
      : "offline";

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

  const dotColor = state === "healthy" ? "bg-success" : "bg-danger";
  const label =
    state === "healthy"
      ? t("header.health_healthy")
      : state === "offline"
        ? t("header.health_offline")
        : t("header.health_checking");

  return (
    <Card className="max-w-md">
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Activity className="size-5 text-muted-foreground" />
          {t("system.health_title")}
        </CardTitle>
        <CardDescription>{t("system.health_desc")}</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="flex items-center gap-3">
          <span className="relative flex size-3">
            {state === "healthy" ? (
              <span
                className={cn(
                  "absolute inline-flex size-full rounded-full opacity-60",
                  "animate-[ppap-pulse-ring_1.8s_cubic-bezier(0.4,0,0.6,1)_infinite]",
                  dotColor,
                )}
              />
            ) : null}
            <span className={cn("relative inline-flex size-3 rounded-full", dotColor)} />
          </span>
          <span className="text-sm font-medium">{label}</span>
          <span className="text-xs text-muted-foreground">
            {query.data?.status ? `(${query.data.status})` : ""}
          </span>
        </div>
        <p className="mt-3 text-xs text-muted-foreground">{t("system.health_polling")}</p>
      </CardContent>
    </Card>
  );
}
