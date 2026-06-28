import * as React from "react";
import { type ColumnDef } from "@tanstack/react-table";
import { RefreshCw } from "lucide-react";
import { useTranslation } from "react-i18next";
import {
  type ApiKeyUsageEntry,
  useApiKeyUsageQuery,
} from "@/lib/api/usage";
import { formatCompact, formatNumber, formatPercent } from "@/lib/utils";
import { Button, DataTable } from "@/components/ui";

/** Flattened row combining the key string with its usage stats. */
interface ApiKeyUsageRow extends ApiKeyUsageEntry {
  key: string;
}

/**
 * API Key Usage tab — displays a simple table of per-downstream-API-key
 * consumption metrics fetched from GET /api-key-usage.
 */
export function ApiKeyUsageTab() {
  const { t, i18n } = useTranslation();
  const locale = i18n.resolvedLanguage ?? i18n.language;
  const query = useApiKeyUsageQuery();

  const rows = React.useMemo<ApiKeyUsageRow[]>(() => {
    if (!query.data) return [];
    return Object.entries(query.data).map(([key, entry]) => ({ key, ...entry }));
  }, [query.data]);

  const columns = React.useMemo<ColumnDef<ApiKeyUsageRow>[]>(
    () => [
      {
        accessorKey: "key",
        header: t("api_key_usage.col_key"),
        cell: ({ row }) => (
          <span className="font-mono text-xs">{row.original.key}</span>
        ),
      },
      {
        accessorKey: "requests",
        header: t("api_key_usage.col_requests"),
        cell: ({ row }) => (
          <span className="tabular-nums">{formatNumber(row.original.requests, locale)}</span>
        ),
      },
      {
        accessorKey: "total_tokens",
        header: t("api_key_usage.col_tokens"),
        cell: ({ row }) => (
          <span className="tabular-nums">{formatCompact(row.original.total_tokens, locale)}</span>
        ),
      },
      {
        accessorKey: "total_input_tokens",
        header: t("api_key_usage.col_input_tokens"),
        cell: ({ row }) => (
          <span className="tabular-nums">
            {formatCompact(row.original.total_input_tokens, locale)}
          </span>
        ),
      },
      {
        accessorKey: "cache_hit_rate",
        header: t("api_key_usage.col_cache_hit"),
        cell: ({ row }) => (
          <span className="tabular-nums">{formatPercent(row.original.cache_hit_rate, locale)}</span>
        ),
      },
      {
        accessorKey: "average_latency_ms",
        header: t("api_key_usage.col_latency"),
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
        header: t("api_key_usage.col_tps"),
        cell: ({ row }) => (
          <span className="tabular-nums">
            {new Intl.NumberFormat(locale, { maximumFractionDigits: 1 }).format(row.original.tps)}
          </span>
        ),
      },
    ],
    [t, locale],
  );

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">{t("api_key_usage.subtitle")}</p>
        <Button variant="outline" size="sm" onClick={() => void query.refetch()}>
          <RefreshCw className="size-4" />
          {t("common.refresh")}
        </Button>
      </div>

      <DataTable
        columns={columns}
        data={rows}
        filterColumnId="key"
        filterPlaceholder={t("api_key_usage.filter_placeholder")}
        loading={query.isLoading}
        error={query.isError}
        emptyTitle={t("api_key_usage.empty_title")}
        emptyDescription={t("api_key_usage.empty_desc")}
        enableColumnVisibility={false}
      />
    </div>
  );
}
