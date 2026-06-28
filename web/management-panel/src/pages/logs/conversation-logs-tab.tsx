import * as React from "react";
import { type ColumnDef } from "@tanstack/react-table";
import { Eye, RefreshCw } from "lucide-react";
import { useTranslation } from "react-i18next";
import {
  type ConversationLogEntry,
  useConversationLogsQuery,
  useConversationLogDetailQuery,
} from "@/lib/api/logs";
import { formatBytes, formatDateTime } from "@/lib/utils";
import {
  Button,
  DataTable,
  Drawer,
  DrawerBody,
  DrawerContent,
  DrawerHeader,
  DrawerTitle,
  DrawerDescription,
  Skeleton,
} from "@/components/ui";

/**
 * Conversation Logs tab (PPAP private feature) — lists conversation log
 * entries with per-key filtering and a detail drawer that tails individual
 * log content.
 */
export function ConversationLogsTab() {
  const { t, i18n } = useTranslation();
  const locale = i18n.resolvedLanguage ?? i18n.language;
  const query = useConversationLogsQuery();
  const [viewTarget, setViewTarget] = React.useState<string | null>(null);
  const detailQuery = useConversationLogDetailQuery(viewTarget);

  const entries = query.data ?? [];

  const columns = React.useMemo<ColumnDef<ConversationLogEntry>[]>(
    () => [
      {
        accessorKey: "id",
        header: t("logs.col_id"),
        cell: ({ row }) => (
          <span className="font-mono text-xs">{row.original.id}</span>
        ),
      },
      {
        accessorKey: "api_key",
        header: t("logs.col_api_key"),
        cell: ({ row }) => (
          <span className="font-mono text-xs">{row.original.api_key}</span>
        ),
      },
      {
        accessorKey: "model",
        header: t("logs.col_model"),
        cell: ({ row }) => (
          <span className="text-xs">{row.original.model}</span>
        ),
      },
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
        accessorKey: "size_bytes",
        header: t("logs.col_size"),
        cell: ({ row }) => (
          <span className="text-xs tabular-nums text-muted-foreground">
            {formatBytes(row.original.size_bytes, locale)}
          </span>
        ),
      },
      {
        id: "actions",
        header: "",
        enableHiding: false,
        enableSorting: false,
        cell: ({ row }) => (
          <div className="flex justify-end">
            <Button
              variant="ghost"
              size="icon-sm"
              aria-label={t("logs.view_log")}
              onClick={() => setViewTarget(row.original.id)}
            >
              <Eye className="size-4" />
            </Button>
          </div>
        ),
      },
    ],
    [t, locale],
  );

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">{t("logs.conversation_subtitle")}</p>
        <Button variant="outline" size="sm" onClick={() => void query.refetch()}>
          <RefreshCw className="size-4" />
          {t("common.refresh")}
        </Button>
      </div>

      <DataTable
        columns={columns}
        data={entries}
        filterColumnId="api_key"
        filterPlaceholder={t("logs.filter_key_placeholder")}
        loading={query.isLoading}
        error={query.isError}
        emptyTitle={t("logs.conversation_empty_title")}
        emptyDescription={t("logs.conversation_empty_desc")}
        pageSize={15}
      />

      {/* Detail drawer */}
      <Drawer open={viewTarget !== null} onOpenChange={(open) => !open && setViewTarget(null)}>
        <DrawerContent className="max-w-2xl">
          <DrawerHeader>
            <DrawerTitle>{t("logs.detail_title")}</DrawerTitle>
            <DrawerDescription>
              {viewTarget ? t("logs.detail_desc", { id: viewTarget }) : ""}
            </DrawerDescription>
          </DrawerHeader>
          <DrawerBody>
            {detailQuery.isLoading ? (
              <div className="space-y-2">
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-4 w-3/4" />
                <Skeleton className="h-4 w-5/6" />
              </div>
            ) : detailQuery.isError ? (
              <p className="text-sm text-danger">{t("logs.detail_error")}</p>
            ) : (
              <pre className="max-h-[60vh] overflow-auto whitespace-pre-wrap rounded-lg border border-border bg-surface-sunken/40 p-4 font-mono text-xs leading-relaxed text-foreground">
                {detailQuery.data ?? ""}
              </pre>
            )}
          </DrawerBody>
        </DrawerContent>
      </Drawer>
    </div>
  );
}
