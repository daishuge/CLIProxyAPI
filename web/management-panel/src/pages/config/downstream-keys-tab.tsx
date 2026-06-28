import * as React from "react";
import { KeyRound, RefreshCw } from "lucide-react";
import { useTranslation } from "react-i18next";
import { ApiError } from "@/lib/api";
import { useDownstreamApiKeysQuery, usePutDownstreamApiKeysMutation } from "@/lib/api/config";
import { StringListField } from "@/components/forms/string-list-field";
import {
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

/** Manage the downstream API key list using the shared StringListField. */
export function DownstreamKeysTab() {
  const { t } = useTranslation();
  const query = useDownstreamApiKeysQuery();
  const mutation = usePutDownstreamApiKeysMutation();

  const [keys, setKeys] = React.useState<string[]>([]);
  const [baseline, setBaseline] = React.useState<string[]>([]);

  React.useEffect(() => {
    if (query.data) {
      const list = query.data["api-keys"] ?? [];
      setKeys(list);
      setBaseline(list);
    }
  }, [query.data]);

  const dirty = React.useMemo(
    () => JSON.stringify(keys) !== JSON.stringify(baseline),
    [keys, baseline],
  );

  const save = () => {
    // Filter out empty strings before saving.
    const filtered = keys.filter((k) => k.trim().length > 0);
    mutation.mutate(filtered, {
      onSuccess: () => {
        setBaseline(filtered);
        setKeys(filtered);
        toast.success(t("config.keys_saved"));
      },
      onError: (error) => {
        const msg = error instanceof ApiError ? error.message : t("common.unknown_error");
        toast.error(t("config.keys_save_failed"), msg);
      },
    });
  };

  if (query.isLoading) {
    return (
      <div className="space-y-3">
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} className="h-10 w-full max-w-lg" />
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
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <KeyRound className="size-5 text-muted-foreground" />
          {t("config.keys_title")}
        </CardTitle>
        <CardDescription>{t("config.keys_desc")}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="max-w-lg">
          <StringListField
            value={keys}
            onChange={setKeys}
            placeholder={t("config.keys_placeholder")}
            addLabel={t("config.keys_add")}
          />
        </div>
        <div className="flex items-center gap-2">
          <Button onClick={save} loading={mutation.isPending} disabled={!dirty}>
            {t("common.save")}
          </Button>
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
      </CardContent>
    </Card>
  );
}
