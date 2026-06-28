import * as React from "react";
import { Database, HardDrive, RefreshCw, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { ApiError } from "@/lib/api";
import { useLogStorageQuery, useDeleteLogsMutation } from "@/lib/api/logs";
import { formatBytes, formatNumber } from "@/lib/utils";
import { StatCard } from "@/components/dashboard/stat-card";
import { Button, toast } from "@/components/ui";
import { ConfirmDialog } from "@/components/forms/confirm-dialog";

/** Supported log types for the delete endpoint. */
const LOG_TYPES = ["request", "error", "conversation"] as const;

/**
 * Log Storage tab (PPAP private feature) — displays storage consumption
 * stats and provides per-type log deletion with a confirmation dialog.
 */
export function LogStorageTab() {
  const { t, i18n } = useTranslation();
  const locale = i18n.resolvedLanguage ?? i18n.language;
  const query = useLogStorageQuery();
  const deleteMutation = useDeleteLogsMutation();

  const [deleteType, setDeleteType] = React.useState<string | null>(null);

  const storage = query.data;
  const loading = query.isLoading;

  const confirmDelete = () => {
    if (!deleteType) return;
    deleteMutation.mutate(deleteType, {
      onSuccess: () => {
        toast.success(t("log_storage.deleted", { type: deleteType }));
        setDeleteType(null);
      },
      onError: (error) => {
        const message = error instanceof ApiError ? error.message : t("common.unknown_error");
        toast.error(t("log_storage.delete_failed"), message);
      },
    });
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">{t("log_storage.subtitle")}</p>
        <Button variant="outline" size="sm" onClick={() => void query.refetch()}>
          <RefreshCw className="size-4" />
          {t("common.refresh")}
        </Button>
      </div>

      {/* Storage stat cards */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <StatCard
          label={t("log_storage.total_size")}
          value={storage ? formatBytes(storage.total_size_bytes, locale) : "—"}
          icon={HardDrive}
          loading={loading}
        />
        <StatCard
          label={t("log_storage.file_count")}
          value={storage ? formatNumber(storage.file_count, locale) : "—"}
          icon={Database}
          loading={loading}
        />
      </div>

      {/* Delete controls per log type */}
      <div className="space-y-3">
        <p className="text-sm font-medium text-foreground">{t("log_storage.cleanup_title")}</p>
        <div className="flex flex-wrap gap-2">
          {LOG_TYPES.map((type) => (
            <Button
              key={type}
              variant="outline"
              size="sm"
              onClick={() => setDeleteType(type)}
            >
              <Trash2 className="size-4" />
              {t(`log_storage.delete_${type}`)}
            </Button>
          ))}
        </div>
      </div>

      <ConfirmDialog
        open={deleteType !== null}
        onOpenChange={(open) => !open && setDeleteType(null)}
        title={t("log_storage.delete_title")}
        description={t("log_storage.delete_desc", { type: deleteType ?? "" })}
        loading={deleteMutation.isPending}
        onConfirm={confirmDelete}
      />
    </div>
  );
}
