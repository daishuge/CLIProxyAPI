import * as React from "react";
import { RefreshCw, FileCode } from "lucide-react";
import { useTranslation } from "react-i18next";
import { ApiError } from "@/lib/api";
import { useConfigYamlQuery, usePutConfigYamlMutation } from "@/lib/api/config";
import { YamlEditor } from "@/components/editor/yaml-editor";
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

/** Raw YAML config editor powered by CodeMirror 6. */
export function YamlEditorTab() {
  const { t } = useTranslation();
  const query = useConfigYamlQuery();
  const mutation = usePutConfigYamlMutation();

  // Local draft mirrors the server YAML; edits are tracked for dirty check.
  const [draft, setDraft] = React.useState("");
  const [baseline, setBaseline] = React.useState("");

  React.useEffect(() => {
    if (query.data !== undefined) {
      setDraft(query.data);
      setBaseline(query.data);
    }
  }, [query.data]);

  const dirty = draft !== baseline;

  const save = () => {
    mutation.mutate(draft, {
      onSuccess: () => {
        setBaseline(draft);
        toast.success(t("config.yaml_saved"));
      },
      onError: (error) => {
        const msg = error instanceof ApiError ? error.message : t("common.unknown_error");
        toast.error(t("config.yaml_save_failed"), msg);
      },
    });
  };

  if (query.isLoading) {
    return <Skeleton className="h-[500px] w-full" />;
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
          <FileCode className="size-5 text-muted-foreground" />
          {t("config.yaml_title")}
        </CardTitle>
        <CardDescription>{t("config.yaml_desc")}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="min-h-[500px] overflow-hidden rounded-md border border-border">
          <YamlEditor value={draft} onChange={setDraft} />
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
