import * as React from "react";
import { Pencil, Plus, RefreshCw, Trash2, Waypoints } from "lucide-react";
import { useTranslation } from "react-i18next";
import { ApiError } from "@/lib/api";
import {
  useDeleteModelAliasChannelMutation,
  useModelAliasesQuery,
} from "@/lib/api/model-aliases";
import {
  useForceModelPrefixQuery,
  useUpdateForceModelPrefixMutation,
} from "@/lib/api/models";
import {
  Badge,
  Button,
  Card,
  CardContent,
  EmptyState,
  Skeleton,
  Switch,
  toast,
} from "@/components/ui";
import { ConfirmDialog } from "@/components/forms/confirm-dialog";
import { ModelAliasDrawer } from "./model-alias-drawer";

/**
 * Model Aliases management. Lists alias channels (each mapping client-facing
 * aliases to upstream model names) with per-channel edit/delete, plus the
 * global force-model-prefix toggle. A maintainable list/editor replaces the
 * legacy relationship-graph UI.
 */
export function ModelAliasesTab() {
  const { t } = useTranslation();
  const aliasQuery = useModelAliasesQuery();
  const deleteMutation = useDeleteModelAliasChannelMutation();
  const prefixQuery = useForceModelPrefixQuery();
  const prefixMutation = useUpdateForceModelPrefixMutation();

  const [drawerOpen, setDrawerOpen] = React.useState(false);
  const [editingChannel, setEditingChannel] = React.useState<string | null>(null);
  const [deleteChannel, setDeleteChannel] = React.useState<string | null>(null);

  const aliasMap = aliasQuery.data ?? {};
  const channels = Object.keys(aliasMap).sort();

  const openCreate = () => {
    setEditingChannel(null);
    setDrawerOpen(true);
  };
  const openEdit = (channel: string) => {
    setEditingChannel(channel);
    setDrawerOpen(true);
  };

  const togglePrefix = (next: boolean) => {
    prefixMutation.mutate(next, {
      onSuccess: () => toast.success(t("model_aliases.prefix_saved")),
      onError: (error) => {
        const message = error instanceof ApiError ? error.message : t("common.unknown_error");
        toast.error(t("model_aliases.prefix_save_failed"), message);
      },
    });
  };

  const confirmDelete = () => {
    if (!deleteChannel) return;
    deleteMutation.mutate(deleteChannel, {
      onSuccess: () => {
        toast.success(t("model_aliases.deleted"));
        setDeleteChannel(null);
      },
      onError: (error) => {
        const message = error instanceof ApiError ? error.message : t("common.unknown_error");
        toast.error(t("model_aliases.delete_failed"), message);
      },
    });
  };

  return (
    <div className="space-y-4">
      {/* Global force-model-prefix toggle */}
      <Card>
        <CardContent className="flex items-center justify-between gap-4 p-4">
          <div>
            <p className="text-sm font-semibold text-foreground">
              {t("model_aliases.force_prefix_title")}
            </p>
            <p className="text-xs text-muted-foreground">
              {t("model_aliases.force_prefix_desc")}
            </p>
          </div>
          {prefixQuery.isLoading ? (
            <Skeleton className="h-5 w-9" />
          ) : (
            <Switch
              checked={prefixQuery.data ?? false}
              onCheckedChange={togglePrefix}
              disabled={prefixMutation.isPending || prefixQuery.isError}
              aria-label={t("model_aliases.force_prefix_title")}
            />
          )}
        </CardContent>
      </Card>

      <div className="flex items-center justify-between">
        <p className="text-sm font-medium text-foreground">{t("model_aliases.channels_title")}</p>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => void aliasQuery.refetch()}>
            <RefreshCw className="size-4" />
            {t("common.refresh")}
          </Button>
          <Button size="sm" onClick={openCreate}>
            <Plus className="size-4" />
            {t("model_aliases.add_channel")}
          </Button>
        </div>
      </div>

      {aliasQuery.isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-16 w-full" />
          ))}
        </div>
      ) : aliasQuery.isError ? (
        <EmptyState
          tone="danger"
          title={t("table.error_title")}
          description={t("table.error_description")}
          action={
            <Button variant="outline" size="sm" onClick={() => void aliasQuery.refetch()}>
              <RefreshCw className="size-4" />
              {t("common.retry")}
            </Button>
          }
        />
      ) : channels.length === 0 ? (
        <EmptyState
          icon={Waypoints}
          title={t("model_aliases.empty_title")}
          description={t("model_aliases.empty_desc")}
          action={
            <Button size="sm" onClick={openCreate}>
              <Plus className="size-4" />
              {t("model_aliases.add_channel")}
            </Button>
          }
        />
      ) : (
        <div className="space-y-2">
          {channels.map((channel) => {
            const aliases = aliasMap[channel] ?? [];
            return (
              <Card key={channel}>
                <CardContent className="flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between">
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="font-medium capitalize text-foreground">{channel}</span>
                      <Badge variant="outline">
                        {t("model_aliases.alias_count", { count: aliases.length })}
                      </Badge>
                    </div>
                    <p className="mt-1 truncate font-mono text-xs text-muted-foreground">
                      {aliases.map((a) => a.alias).join(", ") || "—"}
                    </p>
                  </div>
                  <div className="flex shrink-0 items-center gap-1">
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      onClick={() => openEdit(channel)}
                      aria-label={t("common.edit")}
                    >
                      <Pencil className="size-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      onClick={() => setDeleteChannel(channel)}
                      aria-label={t("common.delete")}
                    >
                      <Trash2 className="size-4 text-danger" />
                    </Button>
                  </div>
                </CardContent>
              </Card>
            );
          })}
        </div>
      )}

      <ModelAliasDrawer
        open={drawerOpen}
        onOpenChange={setDrawerOpen}
        channel={editingChannel}
        aliasMap={aliasMap}
      />

      <ConfirmDialog
        open={deleteChannel !== null}
        onOpenChange={(open) => !open && setDeleteChannel(null)}
        title={t("model_aliases.delete_title")}
        description={t("model_aliases.delete_desc", { channel: deleteChannel ?? "" })}
        loading={deleteMutation.isPending}
        onConfirm={confirmDelete}
      />
    </div>
  );
}
