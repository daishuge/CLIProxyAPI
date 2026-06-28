import * as React from "react";
import { type ColumnDef } from "@tanstack/react-table";
import { RefreshCw } from "lucide-react";
import { useTranslation } from "react-i18next";
import {
  type ErrorLogEntry,
  useErrorLogsQuery,
} from "@/lib/api/logs";
import { formatDateTime } from "@/lib/utils";
import { Badge, type BadgeProps, Button, DataTable } from "@/components/ui";

/** Map HTTP status to a badge variant for error rows. */
function statusVariant(status: number): NonNullable<BadgeProps["variant"]> {
  if (status >= 400 && status < 500) return "warning";
  if (status >= 500) return "danger";
  return "outline";
}

/**
 * Error Logs tab — shows a table of request error log entries with a
 * refresh button.
 */
export function ErrorLogsTab() {
  const { t, i18n } = useTranslation();
  const locale = i18n.resolvedLanguage ?? i18n.language;
  const query = useErrorLogsQuery();

  const entries = query.data ?? [];

  const columns = React.useMemo<ColumnDef<ErrorLogEntry>[]>(
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
        accessorKey: "error",
        header: t("logs.col_error"),
        cell: ({ row }) => (
          <span className="max-w-[320px] truncate text-xs text-danger">{row.original.error}</span>
        ),
      },
    ],
    [t, locale],
  );

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">{t("logs.error_subtitle")}</p>
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
        emptyTitle={t("logs.error_empty_title")}
        emptyDescription={t("logs.error_empty_desc")}
        pageSize={15}
      />
    </div>
  );
}
