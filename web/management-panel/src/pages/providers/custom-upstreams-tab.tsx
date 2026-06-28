import * as React from "react";
import { type ColumnDef } from "@tanstack/react-table";
import { MoreHorizontal, Pencil, Plus, RefreshCw, Server, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { ApiError } from "@/lib/api";
import {
  type CustomUpstream,
  useCustomUpstreamsQuery,
  useDeleteCustomUpstreamMutation,
} from "@/lib/api/custom-upstreams";
import {
  Badge,
  Button,
  Card,
  CardContent,
  DataTable,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  EmptyState,
  toast,
} from "@/components/ui";
import { ConfirmDialog } from "@/components/forms/confirm-dialog";
import { CustomUpstreamDrawer } from "./custom-upstream-drawer";

/**
 * Custom Upstreams management — PPAP's headline feature. Lists OpenAI-compatible
 * upstreams with their key/model counts and provides full CRUD via a drawer
 * editor and a confirm-guarded delete.
 */
export function CustomUpstreamsTab() {
  const { t } = useTranslation();
  const query = useCustomUpstreamsQuery();
  const deleteMutation = useDeleteCustomUpstreamMutation();

  const [drawerOpen, setDrawerOpen] = React.useState(false);
  const [editing, setEditing] = React.useState<CustomUpstream | null>(null);
  const [deleteTarget, setDeleteTarget] = React.useState<CustomUpstream | null>(null);

  const upstreams = query.data ?? [];

  const openCreate = () => {
    setEditing(null);
    setDrawerOpen(true);
  };
  const openEdit = (upstream: CustomUpstream) => {
    setEditing(upstream);
    setDrawerOpen(true);
  };

  const confirmDelete = () => {
    if (!deleteTarget) return;
    deleteMutation.mutate(deleteTarget.name, {
      onSuccess: () => {
        toast.success(t("custom_upstreams.deleted"));
        setDeleteTarget(null);
      },
      onError: (error) => {
        const message = error instanceof ApiError ? error.message : t("common.unknown_error");
        toast.error(t("custom_upstreams.delete_failed"), message);
      },
    });
  };

  const columns = React.useMemo<ColumnDef<CustomUpstream>[]>(
    () => [
      {
        accessorKey: "name",
        header: t("custom_upstreams.col_name"),
        cell: ({ row }) => (
          <div className="flex items-center gap-2">
            <span className="font-medium text-foreground">{row.original.name}</span>
            {row.original.prefix ? (
              <Badge variant="outline" className="font-mono text-[10px]">
                {row.original.prefix}
              </Badge>
            ) : null}
          </div>
        ),
      },
      {
        accessorKey: "base-url",
        header: t("custom_upstreams.col_base_url"),
        cell: ({ row }) => (
          <span className="font-mono text-xs text-muted-foreground">{row.original["base-url"]}</span>
        ),
      },
      {
        id: "keys",
        header: t("custom_upstreams.col_keys"),
        enableSorting: false,
        cell: ({ row }) => (
          <span className="tabular-nums">{row.original["api-key-entries"]?.length ?? 0}</span>
        ),
      },
      {
        id: "models",
        header: t("custom_upstreams.col_models"),
        enableSorting: false,
        cell: ({ row }) => <span className="tabular-nums">{row.original.models?.length ?? 0}</span>,
      },
      {
        id: "status",
        header: t("custom_upstreams.col_status"),
        enableSorting: false,
        cell: ({ row }) =>
          row.original.disabled ? (
            <Badge variant="default">{t("common.disabled")}</Badge>
          ) : (
            <Badge variant="success" dot>
              {t("common.enabled")}
            </Badge>
          ),
      },
      {
        id: "actions",
        header: "",
        enableHiding: false,
        enableSorting: false,
        cell: ({ row }) => (
          <div className="flex justify-end">
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="icon-sm" aria-label={t("common.edit")}>
                  <MoreHorizontal className="size-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onSelect={() => openEdit(row.original)}>
                  <Pencil className="size-4" />
                  {t("common.edit")}
                </DropdownMenuItem>
                <DropdownMenuItem destructive onSelect={() => setDeleteTarget(row.original)}>
                  <Trash2 className="size-4" />
                  {t("common.delete")}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        ),
      },
    ],
    [t],
  );

  return (
    <div className="space-y-4">
      <Card>
        <CardContent className="flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-center gap-3">
            <span className="flex size-9 items-center justify-center rounded-lg bg-brand-500/10 text-brand-600 dark:text-brand-300">
              <Server className="size-5" />
            </span>
            <div>
              <p className="text-sm font-semibold text-foreground">
                {t("custom_upstreams.title")}
              </p>
              <p className="text-xs text-muted-foreground">{t("custom_upstreams.subtitle")}</p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={() => void query.refetch()}>
              <RefreshCw className="size-4" />
              {t("common.refresh")}
            </Button>
            <Button size="sm" onClick={openCreate}>
              <Plus className="size-4" />
              {t("custom_upstreams.add")}
            </Button>
          </div>
        </CardContent>
      </Card>

      {!query.isLoading && !query.isError && upstreams.length === 0 ? (
        <EmptyState
          icon={Server}
          title={t("custom_upstreams.empty_title")}
          description={t("custom_upstreams.empty_desc")}
          action={
            <Button size="sm" onClick={openCreate}>
              <Plus className="size-4" />
              {t("custom_upstreams.add")}
            </Button>
          }
        />
      ) : (
        <DataTable
          columns={columns}
          data={upstreams}
          filterColumnId="name"
          filterPlaceholder={t("custom_upstreams.filter_placeholder")}
          loading={query.isLoading}
          error={query.isError}
          enableColumnVisibility={false}
        />
      )}

      <CustomUpstreamDrawer
        open={drawerOpen}
        onOpenChange={setDrawerOpen}
        upstream={editing}
        existing={upstreams}
      />

      <ConfirmDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title={t("custom_upstreams.delete_title")}
        description={t("custom_upstreams.delete_desc", { name: deleteTarget?.name ?? "" })}
        loading={deleteMutation.isPending}
        onConfirm={confirmDelete}
      />
    </div>
  );
}
