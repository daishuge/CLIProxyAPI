import * as React from "react";
import { type ColumnDef } from "@tanstack/react-table";
import { RefreshCw } from "lucide-react";
import { useTranslation } from "react-i18next";
import {
  type CatalogModel,
  MODEL_CHANNELS,
  type ModelChannel,
  useModelCatalogQueries,
} from "@/lib/api/models";
import {
  Badge,
  Button,
  DataTable,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui";

const ALL_CHANNELS = "__all__";

/**
 * Model catalog browser. Fans out across every channel's static model
 * definitions and renders them in a filterable, searchable table.
 */
export function ModelsTab() {
  const { t } = useTranslation();
  const { models, isLoading, isError, refetch } = useModelCatalogQueries();
  const [channel, setChannel] = React.useState<string>(ALL_CHANNELS);

  const filtered = React.useMemo(
    () => (channel === ALL_CHANNELS ? models : models.filter((m) => m.channel === channel)),
    [models, channel],
  );

  const columns = React.useMemo<ColumnDef<CatalogModel>[]>(
    () => [
      {
        accessorKey: "id",
        header: t("models.col_id"),
        cell: ({ row }) => (
          <span className="font-mono text-xs text-foreground">{row.original.id}</span>
        ),
      },
      {
        accessorKey: "display_name",
        header: t("models.col_display_name"),
        cell: ({ row }) => (
          <span className="text-sm text-muted-foreground">{row.original.display_name ?? "—"}</span>
        ),
      },
      {
        accessorKey: "channel",
        header: t("models.col_channel"),
        cell: ({ row }) => (
          <Badge variant="outline" className="capitalize">
            {row.original.channel}
          </Badge>
        ),
      },
      {
        accessorKey: "owned_by",
        header: t("models.col_owned_by"),
        cell: ({ row }) => (
          <span className="text-sm text-muted-foreground">{row.original.owned_by ?? "—"}</span>
        ),
      },
    ],
    [t],
  );

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <Select value={channel} onValueChange={setChannel}>
          <SelectTrigger className="w-full sm:w-56">
            <SelectValue placeholder={t("models.all_channels")} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value={ALL_CHANNELS}>{t("models.all_channels")}</SelectItem>
            {MODEL_CHANNELS.map((c: ModelChannel) => (
              <SelectItem key={c} value={c} className="capitalize">
                {c}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Button variant="outline" size="sm" onClick={refetch}>
          <RefreshCw className="size-4" />
          {t("common.refresh")}
        </Button>
      </div>

      <DataTable
        columns={columns}
        data={filtered}
        filterColumnId="id"
        filterPlaceholder={t("models.filter_placeholder")}
        loading={isLoading}
        error={isError}
        emptyTitle={t("models.empty_title")}
        emptyDescription={t("models.empty_desc")}
        pageSize={15}
      />
    </div>
  );
}
