import * as React from "react";
import { Download, RefreshCw, Store } from "lucide-react";
import { useTranslation } from "react-i18next";
import { ApiError } from "@/lib/api";
import { usePluginStoreQuery, useInstallPluginMutation, type StorePlugin } from "@/lib/api/plugins";
import {
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  EmptyState,
  Skeleton,
  toast,
} from "@/components/ui";

/** Single store plugin card with an install button. */
function StorePluginCard({
  plugin,
  onInstall,
  installing,
}: {
  plugin: StorePlugin;
  onInstall: (name: string) => void;
  installing: boolean;
}) {
  const { t } = useTranslation();

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="flex items-center gap-2 text-base">
          <Store className="size-4 text-muted-foreground" />
          {plugin.name}
          <Badge variant="outline" className="font-mono text-xs">
            {plugin.version}
          </Badge>
        </CardTitle>
        {plugin.description ? (
          <CardDescription>{plugin.description}</CardDescription>
        ) : null}
      </CardHeader>
      <CardContent>
        {plugin.installed ? (
          <Badge variant="outline">{t("plugins.already_installed")}</Badge>
        ) : (
          <Button
            size="sm"
            onClick={() => onInstall(plugin.name)}
            loading={installing}
            disabled={installing}
          >
            <Download className="size-4" />
            {t("plugins.install")}
          </Button>
        )}
      </CardContent>
    </Card>
  );
}

/** Plugin store — browsable grid of available plugins. */
export function PluginStoreTab() {
  const { t } = useTranslation();
  const query = usePluginStoreQuery();
  const installMutation = useInstallPluginMutation();

  // Track which plugin is being installed for button loading state.
  const [installingName, setInstallingName] = React.useState<string | null>(null);

  const handleInstall = (name: string) => {
    setInstallingName(name);
    installMutation.mutate(name, {
      onSuccess: () => {
        toast.success(t("plugins.install_ok"));
      },
      onError: (error) => {
        const msg = error instanceof ApiError ? error.message : t("common.unknown_error");
        toast.error(t("plugins.install_failed"), msg);
      },
      onSettled: () => setInstallingName(null),
    });
  };

  const plugins = query.data?.plugins ?? [];

  if (query.isLoading) {
    return (
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {Array.from({ length: 6 }).map((_, i) => (
          <Skeleton key={i} className="h-32 w-full" />
        ))}
      </div>
    );
  }

  if (query.isError) {
    return (
      <EmptyState
        tone="danger"
        title={t("table.error_title")}
        description={t("table.error_description")}
        action={
          <Button variant="outline" size="sm" onClick={() => void query.refetch()}>
            <RefreshCw className="size-4" />
            {t("common.retry")}
          </Button>
        }
      />
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex justify-end">
        <Button
          variant="outline"
          size="sm"
          onClick={() => void query.refetch()}
          disabled={query.isFetching}
        >
          <RefreshCw className="size-4" />
          {t("common.refresh")}
        </Button>
      </div>

      {plugins.length === 0 ? (
        <EmptyState
          icon={Store}
          title={t("plugins.store_empty_title")}
          description={t("plugins.store_empty_desc")}
        />
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {plugins.map((plugin) => (
            <StorePluginCard
              key={plugin.name}
              plugin={plugin}
              onInstall={handleInstall}
              installing={installingName === plugin.name}
            />
          ))}
        </div>
      )}
    </div>
  );
}
