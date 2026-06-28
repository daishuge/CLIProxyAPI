import * as React from "react";
import { type ColumnDef } from "@tanstack/react-table";
import {
  Activity,
  CheckCircle2,
  Coins,
  Database,
  Download,
  Gauge,
  RefreshCw,
  Timer,
  Upload,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import { ApiError } from "@/lib/api";
import {
  useUsageStatisticsQuery,
  useExportUsageStatisticsMutation,
  useImportUsageStatisticsMutation,
} from "@/lib/api/usage";
import type { ApiSnapshot } from "@/lib/api/types";
import { downloadBlob, formatCompact, formatNumber, formatPercent } from "@/lib/utils";
import { StatCard } from "@/components/dashboard/stat-card";
import {
  Button,
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  DataTable,
  toast,
} from "@/components/ui";

/** Flattened row for the per-API-key table. */
interface ApiKeyRow {
  key: string;
  total_requests: number;
  total_tokens: number;
  average_latency_ms: number;
  tps: number;
  cache_hit_rate: number;
}

/** Pure SVG bar chart for time-series data. */
function BarChart({
  data,
  label,
  formatter,
  locale,
}: {
  data: Record<string, number>;
  label: string;
  formatter: (v: number, l?: string) => string;
  locale?: string;
}) {
  const entries = Object.entries(data).sort(([a], [b]) => a.localeCompare(b));
  if (entries.length === 0) return null;

  const maxVal = Math.max(...entries.map(([, v]) => v), 1);
  const barWidth = Math.max(4, Math.min(24, Math.floor(600 / entries.length) - 2));
  const chartWidth = entries.length * (barWidth + 2) + 40;
  const chartHeight = 160;
  const topPadding = 24;
  const bottomPadding = 24;
  const leftPadding = 8;
  const barArea = chartHeight - topPadding - bottomPadding;

  return (
    <div className="space-y-2">
      <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">{label}</p>
      <div className="overflow-x-auto rounded-lg border border-border bg-surface-sunken/40 p-3">
        <svg
          viewBox={`0 0 ${chartWidth} ${chartHeight}`}
          className="w-full"
          style={{ minWidth: Math.min(chartWidth, 300) }}
          role="img"
          aria-label={label}
        >
          {/* Bars */}
          {entries.map(([key, value], i) => {
            const barH = (value / maxVal) * barArea;
            const x = leftPadding + i * (barWidth + 2);
            const y = topPadding + barArea - barH;
            return (
              <g key={key}>
                <rect
                  x={x}
                  y={y}
                  width={barWidth}
                  height={Math.max(barH, 1)}
                  rx={2}
                  className="fill-brand-500/70"
                />
                {/* Tooltip-style title */}
                <title>{`${key}: ${formatter(value, locale)}`}</title>
                {/* X-axis label (show every Nth to avoid crowding) */}
                {entries.length <= 14 || i % Math.ceil(entries.length / 14) === 0 ? (
                  <text
                    x={x + barWidth / 2}
                    y={chartHeight - 4}
                    textAnchor="middle"
                    className="fill-muted-foreground"
                    fontSize={8}
                  >
                    {key.length > 5 ? key.slice(-5) : key}
                  </text>
                ) : null}
              </g>
            );
          })}
          {/* Y-axis max label */}
          <text
            x={leftPadding}
            y={topPadding - 6}
            className="fill-muted-foreground"
            fontSize={9}
          >
            {formatter(maxVal, locale)}
          </text>
        </svg>
      </div>
    </div>
  );
}

/**
 * Usage Statistics tab — shows aggregate KPIs, per-API-key breakdown table,
 * time-series charts, and export/import controls.
 */
export function UsageTab() {
  const { t, i18n } = useTranslation();
  const locale = i18n.resolvedLanguage ?? i18n.language;
  const query = useUsageStatisticsQuery();
  const exportMutation = useExportUsageStatisticsMutation();
  const importMutation = useImportUsageStatisticsMutation();
  const fileInputRef = React.useRef<HTMLInputElement>(null);

  const snapshot = query.data?.usage;
  const loading = query.isLoading;

  const successRate =
    snapshot && snapshot.total_requests > 0
      ? snapshot.success_count / snapshot.total_requests
      : 0;

  // Build per-API-key rows from snapshot.apis
  const apiKeyRows = React.useMemo<ApiKeyRow[]>(() => {
    if (!snapshot?.apis) return [];
    return Object.entries(snapshot.apis).map(([key, api]: [string, ApiSnapshot]) => ({
      key,
      total_requests: api.total_requests,
      total_tokens: api.total_tokens,
      average_latency_ms: api.average_latency_ms,
      tps: api.tps,
      cache_hit_rate: api.cache_hit_rate,
    }));
  }, [snapshot?.apis]);

  const onExport = async () => {
    try {
      const response = await exportMutation.mutateAsync();
      const blob = await response.blob();
      downloadBlob(blob, "usage-statistics.json");
      toast.success(t("usage.export_ok"));
    } catch (error) {
      const message = error instanceof ApiError ? error.message : t("common.unknown_error");
      toast.error(t("usage.export_failed"), message);
    }
  };

  const onImportClick = () => fileInputRef.current?.click();

  const onFileSelected = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) return;
    importMutation.mutate(file, {
      onSuccess: () => toast.success(t("usage.import_ok")),
      onError: (error) => {
        const message = error instanceof ApiError ? error.message : t("common.unknown_error");
        toast.error(t("usage.import_failed"), message);
      },
    });
  };

  const columns = React.useMemo<ColumnDef<ApiKeyRow>[]>(
    () => [
      {
        accessorKey: "key",
        header: t("usage.col_api_key"),
        cell: ({ row }) => (
          <span className="font-mono text-xs">{row.original.key}</span>
        ),
      },
      {
        accessorKey: "total_requests",
        header: t("usage.col_requests"),
        cell: ({ row }) => (
          <span className="tabular-nums">{formatNumber(row.original.total_requests, locale)}</span>
        ),
      },
      {
        accessorKey: "total_tokens",
        header: t("usage.col_tokens"),
        cell: ({ row }) => (
          <span className="tabular-nums">{formatCompact(row.original.total_tokens, locale)}</span>
        ),
      },
      {
        accessorKey: "average_latency_ms",
        header: t("usage.col_latency"),
        cell: ({ row }) => (
          <span className="tabular-nums">
            {t("overview.ms_unit", {
              value: formatNumber(row.original.average_latency_ms, locale),
            })}
          </span>
        ),
      },
      {
        accessorKey: "tps",
        header: t("usage.col_tps"),
        cell: ({ row }) => (
          <span className="tabular-nums">
            {new Intl.NumberFormat(locale, { maximumFractionDigits: 1 }).format(row.original.tps)}
          </span>
        ),
      },
      {
        accessorKey: "cache_hit_rate",
        header: t("usage.col_cache_hit"),
        cell: ({ row }) => (
          <span className="tabular-nums">{formatPercent(row.original.cache_hit_rate, locale)}</span>
        ),
      },
    ],
    [t, locale],
  );

  return (
    <div className="space-y-6">
      {/* Action bar */}
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">{t("usage.subtitle")}</p>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => void query.refetch()}>
            <RefreshCw className="size-4" />
            {t("common.refresh")}
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => void onExport()}
            loading={exportMutation.isPending}
          >
            <Download className="size-4" />
            {t("usage.export")}
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={onImportClick}
            loading={importMutation.isPending}
          >
            <Upload className="size-4" />
            {t("usage.import")}
          </Button>
          <input
            ref={fileInputRef}
            type="file"
            accept=".json,application/json"
            hidden
            onChange={onFileSelected}
          />
        </div>
      </div>

      {/* Stat cards */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
        <StatCard
          label={t("overview.stat_total_requests")}
          value={snapshot ? formatNumber(snapshot.total_requests, locale) : "—"}
          icon={Activity}
          loading={loading}
        />
        <StatCard
          label={t("overview.stat_success_rate")}
          value={snapshot ? formatPercent(successRate, locale) : "—"}
          icon={CheckCircle2}
          tone="success"
          loading={loading}
        />
        <StatCard
          label={t("overview.stat_total_tokens")}
          value={snapshot ? formatCompact(snapshot.total_tokens, locale) : "—"}
          icon={Coins}
          tone="info"
          loading={loading}
        />
        <StatCard
          label={t("overview.stat_avg_latency")}
          value={
            snapshot
              ? t("overview.ms_unit", { value: formatNumber(snapshot.average_latency_ms, locale) })
              : "—"
          }
          icon={Timer}
          loading={loading}
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
          loading={loading}
        />
        <StatCard
          label={t("overview.stat_cache_hit")}
          value={snapshot ? formatPercent(snapshot.cache_hit_rate, locale) : "—"}
          icon={Database}
          tone="info"
          loading={loading}
        />
      </div>

      {/* Per-API-key breakdown */}
      <Card>
        <CardHeader>
          <CardTitle>{t("usage.per_key_title")}</CardTitle>
        </CardHeader>
        <CardContent>
          <DataTable
            columns={columns}
            data={apiKeyRows}
            filterColumnId="key"
            filterPlaceholder={t("usage.filter_key_placeholder")}
            loading={loading}
            error={query.isError}
            emptyTitle={t("usage.per_key_empty_title")}
            emptyDescription={t("usage.per_key_empty_desc")}
            enableColumnVisibility={false}
          />
        </CardContent>
      </Card>

      {/* Time series charts */}
      {snapshot?.requests_by_day && Object.keys(snapshot.requests_by_day).length > 0 ? (
        <Card>
          <CardHeader>
            <CardTitle>{t("usage.charts_title")}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-6">
            {snapshot.requests_by_day ? (
              <BarChart
                data={snapshot.requests_by_day}
                label={t("usage.chart_requests_by_day")}
                formatter={formatNumber}
                locale={locale}
              />
            ) : null}
            {snapshot.tokens_by_day ? (
              <BarChart
                data={snapshot.tokens_by_day}
                label={t("usage.chart_tokens_by_day")}
                formatter={formatCompact}
                locale={locale}
              />
            ) : null}
          </CardContent>
        </Card>
      ) : null}
    </div>
  );
}
