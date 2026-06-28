import * as React from "react";
import { type ColumnDef } from "@tanstack/react-table";
import { RefreshCw } from "lucide-react";
import { useTranslation } from "react-i18next";
import {
  type LogEntry,
  useRequestLogsQuery,
} from "@/lib/api/logs";
import { formatDateTime, formatNumber } from "@/lib/utils";
import { Badge, type BadgeProps, Button, DataTable } from "@/components/ui";

/** Map HTTP status to a visual badge variant. */
function statusVariant(status: number): NonNullable<BadgeProps["variant"]> {
  if (status >= 200 && status < 300) return "success";
  if (status >= 400 && status < 500) return "warning";
  if (status >= 500) return "danger";
  return "outline";
}

/**
 * Request Logs tab — shows a table of recent request log entries.
 * The backend may also return a boolean indicating whether logging is
 * enabled or disabled; in that case we show the current status.
 */
export function RequestLogsTab() {
  const { t, i18n } = useTranslation();
  const locale = i18n.resolvedLanguage ?? i18n.language;
  const query = useRequestLogsQuery();

  // The API may return either an array of logs or a boolean flag.
  const raw = query.data;
  const isBoolean =
    raw !== undefined &&
    typeof (raw as Record<string, unknown>)["request-log"] === "boolean";
  const loggingEnabled = isBoolean
    ? (raw as { "request-log": boolean })["request-log"]
    : true;
  const entries: LogEntry[] = React.useMemo(() => {
    if (!raw) return [];
    const inner = (raw as Record<string, unknown>)["request-log"];
    if (Array.isArray(inner)) return inner as LogEntry[];
    return [];
  }, [raw]);

  const columns = React.useMemo<ColumnDef<LogEntry>[]>(
    () => [
      {
        accessorKey: "timestamp",
        header: t("logs.col_time"),
        cell: ({ row }) => (
          <span className="text-xs text-muted-foreground">
            {formatDateTime(row.original.timestamp, locale)}
          </span>
        ),
      },
      {
        accessorKey: "method",
        header: t("logs.col_method"),
        cell: ({ row }) => (
          <Badge variant="outline" className="font-mono text-xs">
            {row.original.method}
          </Badge>
        ),
      },
      {
        accessorKey: "path",
        header: t("logs.col_path"),
        cell: ({ row }) => (
          <span className="max-w-[240px] truncate font-mono text-xs">{row.original.path}</span>
        ),
      },
      {
        accessorKey: "status",
        header: t("logs.col_status"),
        cell: ({ row }) => (
          <Badge variant={statusVariant(row.original.status)} className="tabular-nums">
            {row.original.status}
          </Badge>
        ),
      },
      {
        accessorKey: "latency_ms",
        header: t("logs.col_latency"),
        cell: ({ row }) => (
          <span className="tabular-nums text-xs text-muted-foreground">
            {t("overview.ms_unit", {
              value: formatNumber(row.original.latency_ms, locale),
            })}
          </span>
        ),
      },
      {
        accessorKey: "model",
        header: t("logs.col_model"),
        cell: ({ row }) =>
          row.original.model ? (
            <span className="text-xs">{row.original.model}</span>
          ) : (
            <span className="text-muted-foreground">—</span>
          ),
      },
      {
        accessorKey: "api_key",
        header: t("logs.col_api_key"),
        cell: ({ row }) =>
          row.original.api_key ? (
            <span className="font-mono text-xs">{row.original.api_key}</span>
          ) : (
            <span className="text-muted-foreground">—</span>
          ),
      },
    ],
    [t, locale],
  );

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <p className="text-sm text-muted-foreground">{t("logs.request_subtitle")}</p>
          {isBoolean ? (
            <Badge variant={loggingEnabled ? "success" : "default"} dot>
              {loggingEnabled ? t("common.enabled") : t("common.disabled")}
            </Badge>
          ) : null}
        </div>
        <Button variant="outline" size="sm" onClick={() => void query.refetch()}>
          <RefreshCw className="size-4" />
          {t("common.refresh")}
        </Button>
      </div>

      <DataTable
        columns={columns}
        data={entries}
        filterColumnId="path"
        filterPlaceholder={t("logs.filter_path_placeholder")}
        loading={query.isLoading}
        error={query.isError}
        emptyTitle={t("logs.request_empty_title")}
        emptyDescription={t("logs.request_empty_desc")}
        pageSize={15}
      />
    </div>
  );
}
