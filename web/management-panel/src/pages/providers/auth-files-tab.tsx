import * as React from "react";
import { type ColumnDef } from "@tanstack/react-table";
import {
  Download,
  Eye,
  MoreHorizontal,
  RefreshCw,
  Trash2,
  Upload,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import { ApiError } from "@/lib/api";
import {
  type AuthFileInfo,
  downloadAuthFile,
  useAuthFilesQuery,
  useDeleteAuthFileMutation,
  useToggleAuthFileStatusMutation,
  useUploadAuthFilesMutation,
} from "@/lib/api/auth-files";
import { downloadBlob, formatBytes, formatDateTime } from "@/lib/utils";
import {
  Badge,
  type BadgeProps,
  Button,
  DataTable,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  Switch,
  toast,
} from "@/components/ui";
import { ConfirmDialog } from "@/components/forms/confirm-dialog";
import { AuthFileModelsDrawer } from "./auth-file-models-drawer";

/** Map an auth status string to a badge variant. */
function statusVariant(status?: string): NonNullable<BadgeProps["variant"]> {
  switch ((status ?? "").toLowerCase()) {
    case "active":
    case "ok":
    case "valid":
      return "success";
    case "disabled":
      return "default";
    case "error":
    case "invalid":
    case "expired":
      return "danger";
    case "cooling":
    case "pending":
      return "warning";
    default:
      return "outline";
  }
}

/**
 * Auth Files management. Lists stored provider credentials with their status,
 * provider, account and size, and supports upload, download, enable/disable,
 * model inspection and deletion.
 */
export function AuthFilesTab() {
  const { t, i18n } = useTranslation();
  const locale = i18n.resolvedLanguage ?? i18n.language;
  const query = useAuthFilesQuery();
  const uploadMutation = useUploadAuthFilesMutation();
  const deleteMutation = useDeleteAuthFileMutation();
  const toggleMutation = useToggleAuthFileStatusMutation();

  const fileInputRef = React.useRef<HTMLInputElement>(null);
  const [deleteTarget, setDeleteTarget] = React.useState<AuthFileInfo | null>(null);
  const [modelsTarget, setModelsTarget] = React.useState<string | null>(null);

  const files = query.data ?? [];

  const onUploadClick = () => fileInputRef.current?.click();

  const onFilesSelected = (event: React.ChangeEvent<HTMLInputElement>) => {
    const selected = Array.from(event.target.files ?? []);
    event.target.value = "";
    if (selected.length === 0) return;
    uploadMutation.mutate(selected, {
      onSuccess: () => toast.success(t("auth_files.upload_ok", { count: selected.length })),
      onError: (error) => {
        const message = error instanceof ApiError ? error.message : t("common.unknown_error");
        toast.error(t("auth_files.upload_failed"), message);
      },
    });
  };

  const onDownload = async (file: AuthFileInfo) => {
    try {
      const response = await downloadAuthFile(file.name);
      const blob = await response.blob();
      downloadBlob(blob, file.name);
    } catch (error) {
      const message = error instanceof ApiError ? error.message : t("common.unknown_error");
      toast.error(t("auth_files.download_failed"), message);
    }
  };

  const onToggle = (file: AuthFileInfo) => {
    toggleMutation.mutate(
      { name: file.name, disabled: !file.disabled },
      {
        onError: (error) => {
          const message = error instanceof ApiError ? error.message : t("common.unknown_error");
          toast.error(t("auth_files.toggle_failed"), message);
        },
      },
    );
  };

  const confirmDelete = () => {
    if (!deleteTarget) return;
    deleteMutation.mutate(deleteTarget.name, {
      onSuccess: () => {
        toast.success(t("auth_files.deleted"));
        setDeleteTarget(null);
      },
      onError: (error) => {
        const message = error instanceof ApiError ? error.message : t("common.unknown_error");
        toast.error(t("auth_files.delete_failed"), message);
      },
    });
  };

  const columns = React.useMemo<ColumnDef<AuthFileInfo>[]>(
    () => [
      {
        accessorKey: "name",
        header: t("auth_files.col_name"),
        cell: ({ row }) => (
          <div className="min-w-0">
            <p className="truncate font-medium text-foreground">{row.original.name}</p>
            {row.original.email || row.original.account ? (
              <p className="truncate text-xs text-muted-foreground">
                {row.original.email ?? row.original.account}
              </p>
            ) : null}
          </div>
        ),
      },
      {
        accessorKey: "provider",
        header: t("auth_files.col_provider"),
        cell: ({ row }) =>
          row.original.provider ? (
            <Badge variant="outline" className="capitalize">
              {row.original.provider}
            </Badge>
          ) : (
            <span className="text-muted-foreground">—</span>
          ),
      },
      {
        accessorKey: "status",
        header: t("auth_files.col_status"),
        cell: ({ row }) => (
          <Badge variant={statusVariant(row.original.status)} dot>
            {row.original.status ?? t("common.unknown")}
          </Badge>
        ),
      },
      {
        id: "enabled",
        header: t("auth_files.col_enabled"),
        enableSorting: false,
        cell: ({ row }) => (
          <Switch
            checked={!row.original.disabled}
            onCheckedChange={() => onToggle(row.original)}
            disabled={toggleMutation.isPending}
            aria-label={t("auth_files.col_enabled")}
          />
        ),
      },
      {
        accessorKey: "size",
        header: t("auth_files.col_size"),
        cell: ({ row }) => (
          <span className="text-xs tabular-nums text-muted-foreground">
            {formatBytes(row.original.size ?? 0, locale)}
          </span>
        ),
      },
      {
        accessorKey: "updated_at",
        header: t("auth_files.col_updated"),
        cell: ({ row }) => (
          <span className="text-xs text-muted-foreground">
            {formatDateTime(row.original.updated_at ?? row.original.modtime, locale)}
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
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="icon-sm" aria-label={t("common.edit")}>
                  <MoreHorizontal className="size-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onSelect={() => setModelsTarget(row.original.name)}>
                  <Eye className="size-4" />
                  {t("auth_files.view_models")}
                </DropdownMenuItem>
                <DropdownMenuItem onSelect={() => void onDownload(row.original)}>
                  <Download className="size-4" />
                  {t("auth_files.download")}
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
    // onToggle/onDownload are stable enough; columns rebuild on locale/lang change.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [t, locale, toggleMutation.isPending],
  );

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">{t("auth_files.subtitle")}</p>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => void query.refetch()}>
            <RefreshCw className="size-4" />
            {t("common.refresh")}
          </Button>
          <Button size="sm" onClick={onUploadClick} loading={uploadMutation.isPending}>
            <Upload className="size-4" />
            {t("auth_files.upload")}
          </Button>
          <input
            ref={fileInputRef}
            type="file"
            accept=".json,application/json"
            multiple
            hidden
            onChange={onFilesSelected}
          />
        </div>
      </div>

      <DataTable
        columns={columns}
        data={files}
        filterColumnId="name"
        filterPlaceholder={t("auth_files.filter_placeholder")}
        loading={query.isLoading}
        error={query.isError}
        emptyTitle={t("auth_files.empty_title")}
        emptyDescription={t("auth_files.empty_desc")}
        pageSize={12}
      />

      <AuthFileModelsDrawer name={modelsTarget} onClose={() => setModelsTarget(null)} />

      <ConfirmDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
        title={t("auth_files.delete_title")}
        description={t("auth_files.delete_desc", { name: deleteTarget?.name ?? "" })}
        loading={deleteMutation.isPending}
        onConfirm={confirmDelete}
      />
    </div>
  );
}
