import * as React from "react";
import { RefreshCw, Puzzle } from "lucide-react";
import { useTranslation } from "react-i18next";
import { ApiError } from "@/lib/api";
import { usePluginsQuery, useUpdatePluginMutation, type Plugin } from "@/lib/api/plugins";
import {
  Button,
  Card,
  CardDescription,
  CardHeader,
  CardTitle,
  Badge,
  EmptyState,
  Skeleton,
  Switch,
  toast,
} from "@/components/ui";

/** Card representing a single installed plugin. */
function PluginCard({
  plugin,
  onToggle,
  toggling,
}: {
  plugin: Plugin;
  onToggle: (name: string, enabled: boolean) => void;
  toggling: boolean;
}) {
  const { t } = useTranslation();

  return (
    <Card>
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between gap-4">
          <div className="space-y-1">
            <CardTitle className="flex items-center gap-2 text-base">
              <Puzzle className="size-4 text-muted-foreground" />
              {plugin.name}
              <Badge variant="outline" className="font-mono text-xs">
                {plugin.version}
              </Badge>
            </CardTitle>
            {plugin.description ? (
              <CardDescription>{plugin.description}</CardDescription>
            ) : null}
          </div>
          <div className="flex items-center gap-2">
            <span className="text-xs text-muted-foreground">
              {plugin.enabled ? t("common.enabled") : t("common.disabled")}
            </span>
            <Switch
              checked={plugin.enabled}
              disabled={toggling}
              onCheckedChange={(checked) => onToggle(plugin.name, checked)}
            />
          </div>
        </div>
      </CardHeader>
    </Card>
  );
}

/** Installed plugins list with enable/disable toggles. */
export function InstalledPluginsTab() {
  const { t } = useTranslation();
  const query = usePluginsQuery();
  const mutation = useUpdatePluginMutation();

  // Track which plugin is currently being toggled to disable its switch.
  const [togglingName, setTogglingName] = React.useState<string | null>(null);

  const handleToggle = (name: string, enabled: boolean) => {
    setTogglingName(name);
    mutation.mutate(
      { name, enabled },
      {
        onSuccess: () => {
          toast.success(t("plugins.toggle_ok"));
        },
        onError: (error) => {
          const msg = error instanceof ApiError ? error.message : t("common.unknown_error");
          toast.error(t("plugins.toggle_failed"), msg);
        },
        onSettled: () => setTogglingName(null),
      },
    );
  };

  const plugins = query.data?.plugins ?? [];

  if (query.isLoading) {
    return (
      <div className="space-y-3">
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} className="h-20 w-full" />
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
          icon={Puzzle}
          title={t("plugins.empty_title")}
          description={t("plugins.empty_desc")}
        />
      ) : (
        <div className="space-y-3">
          {plugins.map((plugin) => (
            <PluginCard
              key={plugin.name}
              plugin={plugin}
              onToggle={handleToggle}
              toggling={togglingName === plugin.name}
            />
          ))}
        </div>
      )}
    </div>
  );
}
