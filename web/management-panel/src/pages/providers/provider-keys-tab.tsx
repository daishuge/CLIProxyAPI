import * as React from "react";
import { type ColumnDef } from "@tanstack/react-table";
import { MoreHorizontal, Pencil, Plus, RefreshCw, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { ApiError } from "@/lib/api";
import {
  type ProviderKey,
  type ProviderKeyType,
  PROVIDER_KEY_TYPES,
  useDeleteProviderKeyMutation,
  useProviderKeysQuery,
} from "@/lib/api/provider-keys";
import { maskSecret } from "@/lib/utils";
import {
  Badge,
  Button,
  DataTable,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  toast,
} from "@/components/ui";
import { ConfirmDialog } from "@/components/forms/confirm-dialog";
import { ProviderKeyDrawer } from "./provider-key-drawer";

/**
 * Provider API Keys management. A provider selector switches between the four
 * key-based families; each renders its keys in a table with create/edit/delete.
 */
export function ProviderKeysTab() {
  const { t } = useTranslation();
  const [provider, setProvider] = React.useState<ProviderKeyType>("claude");

  const query = useProviderKeysQuery(provider);
  const deleteMutation = useDeleteProviderKeyMutation(provider);

  const [drawerOpen, setDrawerOpen] = React.useState(false);
  const [editing, setEditing] = React.useState<ProviderKey | null>(null);
  const [deleteTarget, setDeleteTarget] = React.useState<ProviderKey | null>(null);

  const keys = query.data ?? [];

  const openCreate = () => {
    setEditing(null);
    setDrawerOpen(true);
  };
  const openEdit = (key: ProviderKey) => {
    setEditing(key);
    setDrawerOpen(true);
  };

  const confirmDelete = () => {
    if (!deleteTarget) return;
    deleteMutation.mutate(
      { apiKey: deleteTarget["api-key"], baseUrl: deleteTarget["base-url"] },
      {
        onSuccess: () => {
          toast.success(t("provider_keys.deleted"));
          setDeleteTarget(null);
        },
        onError: (error) => {
          const message = error instanceof ApiError ? error.message : t("common.unknown_error");
          toast.error(t("provider_keys.delete_failed"), message);
        },
      },
    );
  };

  const columns = React.useMemo<ColumnDef<ProviderKey>[]>(
    () => [
      {
        accessorKey: "api-key",
        header: t("provider_keys.col_key"),
        cell: ({ row }) => (
          <span className="font-mono text-xs text-foreground">
            {maskSecret(row.original["api-key"])}
          </span>
        ),
      },
      {
        accessorKey: "base-url",
        header: t("provider_keys.col_base_url"),
        cell: ({ row }) => (
          <span className="font-mono text-xs text-muted-foreground">
            {row.original["base-url"] || t("provider_keys.default_base_url")}
          </span>
        ),
      },
      {
        accessorKey: "prefix",
        header: t("provider_keys.col_prefix"),
        cell: ({ row }) =>
          row.original.prefix ? (
            <Badge variant="outline" className="font-mono text-[10px]">
              {row.original.prefix}
            </Badge>
          ) : (
            <span className="text-muted-foreground">—</span>
          ),
      },
      {
        accessorKey: "priority",
        header: t("provider_keys.col_priority"),
        cell: ({ row }) => (
          <span className="tabular-nums text-muted-foreground">{row.original.priority ?? 0}</span>
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
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <Select value={provider} onValueChange={(v) => setProvider(v as ProviderKeyType)}>
          <SelectTrigger className="w-full sm:w-56">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {PROVIDER_KEY_TYPES.map((p) => (
              <SelectItem key={p} value={p}>
                {t(`provider_keys.provider_${p}`)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => void query.refetch()}>
            <RefreshCw className="size-4" />
            {t("common.refresh")}
          </Button>
          <Button size="sm" onClick={openCreate}>
            <Plus className="size-4" />
            {t("provider_keys.add")}
          </Button>
        </div>
      </div>

      <DataTable
        columns={columns}
        data={keys}
        loading={query.isLoading}
        error={query.isError}
        emptyTitle={t("provider_keys.empty_title")}
        emptyDescription={t("provider_keys.empty_desc")}
        enableColumnVisibility={false}
      />

      <ProviderKeyDrawer
        open={drawerOpen}
        onOpenChange={setDrawerOpen}
        provider={provider}
        providerKey={editing}
        existing={keys}
      />

      <ConfirmDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title={t("provider_keys.delete_title")}
        description={t("provider_keys.delete_desc")}
        loading={deleteMutation.isPending}
        onConfirm={confirmDelete}
      />
    </div>
  );
}
