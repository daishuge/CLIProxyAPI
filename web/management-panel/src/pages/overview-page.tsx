import { useTranslation } from "react-i18next";
import {
  Activity,
  CheckCircle2,
  Coins,
  Database,
  Gauge,
  KeyRound,
  RefreshCw,
  Timer,
  XCircle,
  Server,
} from "lucide-react";
import { ApiError } from "@/lib/api";
import {
  useHealthQuery,
  useUsageStatisticsQuery,
  useApiKeysQuery,
} from "@/lib/hooks";
import { APP_VERSION } from "@/lib/version";
import { formatCompact, formatNumber, formatPercent } from "@/lib/utils";
import { PageHeader } from "@/components/layout/page-header";
import { StatCard } from "@/components/dashboard/stat-card";
import {
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  EmptyState,
} from "@/components/ui";

export function OverviewPage() {
  const { t, i18n } = useTranslation();
  const locale = i18n.resolvedLanguage ?? i18n.language;

  const health = useHealthQuery();
  const usage = useUsageStatisticsQuery();
  const apiKeys = useApiKeysQuery();

  const snapshot = usage.data?.usage;
  const usageDisabled =
    usage.isError && usage.error instanceof ApiError && usage.error.status === 404;
  const usageLoading = usage.isLoading;
  const hasUsage =
    !!snapshot && (snapshot.total_requests > 0 || snapshot.success_count > 0 || snapshot.failure_count > 0);

  const successRate =
    snapshot && snapshot.total_requests > 0
      ? snapshot.success_count / snapshot.total_requests
      : 0;

  const activeKeyCount = apiKeys.data?.["api-keys"]?.length ?? 0;

  const healthy = health.data?.status === "ok" && !health.isError;

  return (
    <div className="space-y-6">
      <PageHeader
        title={t("overview.title")}
        description={t("overview.subtitle")}
        actions={
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              void health.refetch();
              void usage.refetch();
              void apiKeys.refetch();
            }}
          >
            <RefreshCw className="size-4" />
            {t("common.refresh")}
          </Button>
        }
      />

      {/* Backend status banner */}
      <Card>
        <CardContent className="flex flex-col gap-4 p-5 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-center gap-3">
            <span
              className={`flex size-10 items-center justify-center rounded-lg ${
                healthy ? "bg-success/10 text-success" : "bg-danger/10 text-danger"
              }`}
            >
              <Server className="size-5" />
            </span>
            <div>
              <p className="text-sm font-semibold text-foreground">{t("overview.backend_status")}</p>
              <p className="text-xs text-muted-foreground">{t("overview.backend_status_desc")}</p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Badge variant={healthy ? "success" : "danger"} dot>
              {healthy ? t("overview.running") : t("header.health_offline")}
            </Badge>
            <Badge variant="outline" className="font-mono">
              v{APP_VERSION}
            </Badge>
          </div>
        </CardContent>
      </Card>

      {/* Primary KPI grid */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard
          label={t("overview.stat_total_requests")}
          value={snapshot ? formatNumber(snapshot.total_requests, locale) : "—"}
          icon={Activity}
          loading={usageLoading}
        />
        <StatCard
          label={t("overview.stat_success_rate")}
          value={snapshot ? formatPercent(successRate, locale) : "—"}
          icon={CheckCircle2}
          tone="success"
          loading={usageLoading}
        />
        <StatCard
          label={t("overview.stat_total_tokens")}
          value={snapshot ? formatCompact(snapshot.total_tokens, locale) : "—"}
          icon={Coins}
          tone="info"
          loading={usageLoading}
        />
        <StatCard
          label={t("overview.stat_failures")}
          value={snapshot ? formatNumber(snapshot.failure_count, locale) : "—"}
          icon={XCircle}
          tone="danger"
          loading={usageLoading}
        />
        <StatCard
          label={t("overview.stat_cache_hit")}
          value={snapshot ? formatPercent(snapshot.cache_hit_rate, locale) : "—"}
          icon={Database}
          tone="info"
          loading={usageLoading}
        />
        <StatCard
          label={t("overview.stat_avg_latency")}
          value={
            snapshot
              ? t("overview.ms_unit", { value: formatNumber(snapshot.average_latency_ms, locale) })
              : "—"
          }
          icon={Timer}
          loading={usageLoading}
        />
        <StatCard
          label={t("overview.stat_throughput")}
          value={
            snapshot
              ? t("overview.tps_unit", {
                  value: new Intl.NumberFormat(locale, { maximumFractionDigits: 1 }).format(
                    snapshot.tps,
                  ),
                })
              : "—"
          }
          icon={Gauge}
          tone="warning"
          loading={usageLoading}
        />
        <StatCard
          label={t("overview.stat_active_keys")}
          value={apiKeys.isLoading ? "—" : formatNumber(activeKeyCount, locale)}
          icon={KeyRound}
          tone="default"
          loading={apiKeys.isLoading}
        />
      </div>

      {/* Usage metrics panel — proves the three-state pattern end to end */}
      <Card>
        <CardHeader>
          <CardTitle>{t("overview.metrics_title")}</CardTitle>
          <CardDescription>{t("overview.metrics_desc")}</CardDescription>
        </CardHeader>
        <CardContent>
          {usageDisabled ? (
            <EmptyState
              icon={Database}
              title={t("overview.metrics_unavailable_title")}
              description={t("overview.metrics_unavailable_desc")}
            />
          ) : usage.isError ? (
            <EmptyState
              tone="danger"
              icon={XCircle}
              title={t("table.error_title")}
              description={t("table.error_description")}
              action={
                <Button variant="outline" size="sm" onClick={() => void usage.refetch()}>
                  <RefreshCw className="size-4" />
                  {t("common.retry")}
                </Button>
              }
            />
          ) : !usageLoading && !hasUsage ? (
            <EmptyState
              icon={Activity}
              title={t("overview.metrics_empty_title")}
              description={t("overview.metrics_empty_desc")}
            />
          ) : (
            <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
              {[
                {
                  label: t("overview.stat_total_requests"),
                  value: snapshot ? formatNumber(snapshot.total_requests, locale) : "—",
                },
                {
                  label: t("overview.stat_total_tokens"),
                  value: snapshot ? formatCompact(snapshot.total_tokens, locale) : "—",
                },
                {
                  label: t("overview.stat_cache_hit"),
                  value: snapshot ? formatPercent(snapshot.cache_hit_rate, locale) : "—",
                },
                {
                  label: t("overview.stat_avg_latency"),
                  value: snapshot
                    ? t("overview.ms_unit", {
                        value: formatNumber(snapshot.average_latency_ms, locale),
                      })
                    : "—",
                },
              ].map((item) => (
                <div key={item.label} className="rounded-lg border border-border bg-surface-sunken/40 p-4">
                  <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                    {item.label}
                  </p>
                  <p className="mt-1 text-lg font-semibold tabular-nums text-foreground">
                    {item.value}
                  </p>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
