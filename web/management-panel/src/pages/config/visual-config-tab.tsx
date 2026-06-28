import * as React from "react";
import { RefreshCw, Settings } from "lucide-react";
import { useTranslation } from "react-i18next";
import { ApiError } from "@/lib/api";
import { useManagementConfigQuery, usePutConfigMutation } from "@/lib/api/config";
import type { ManagementConfig } from "@/lib/api/types";
import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  EmptyState,
  Input,
  Label,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Skeleton,
  Switch,
  toast,
} from "@/components/ui";

/** Fields safe for visual editing (excludes secret-key and other sensitive entries). */
const SAFE_FIELDS = [
  "debug",
  "proxy-url",
  "request-log",
  "routing-strategy",
  "usage-statistics-enabled",
] as const;

type SafeField = (typeof SAFE_FIELDS)[number];

const ROUTING_STRATEGIES = ["round-robin", "fill-first"] as const;

/** Type of the value for a safe config field. */
function fieldType(field: SafeField): "boolean" | "string" | "select" {
  switch (field) {
    case "debug":
    case "request-log":
    case "usage-statistics-enabled":
      return "boolean";
    case "routing-strategy":
      return "select";
    default:
      return "string";
  }
}

/** Visual config form for key configuration items. */
export function VisualConfigTab() {
  const { t } = useTranslation();
  const query = useManagementConfigQuery();
  const mutation = usePutConfigMutation();

  // Local draft state that mirrors the server config (only safe fields).
  const [draft, setDraft] = React.useState<Partial<ManagementConfig>>({});

  // Seed draft when query data arrives.
  React.useEffect(() => {
    if (query.data) {
      const safe: Partial<ManagementConfig> = {};
      for (const key of SAFE_FIELDS) {
        if (query.data[key] !== undefined) {
          (safe as Record<string, unknown>)[key] = query.data[key];
        }
      }
      setDraft(safe);
    }
  }, [query.data]);

  const dirty = React.useMemo(() => {
    if (!query.data) return false;
    return SAFE_FIELDS.some((key) => draft[key] !== query.data![key]);
  }, [draft, query.data]);

  const updateField = (field: SafeField, value: unknown) => {
    setDraft((prev) => ({ ...prev, [field]: value }));
  };

  const save = () => {
    mutation.mutate(draft, {
      onSuccess: () => toast.success(t("config.visual_saved")),
      onError: (error) => {
        const msg = error instanceof ApiError ? error.message : t("common.unknown_error");
        toast.error(t("config.visual_save_failed"), msg);
      },
    });
  };

  if (query.isLoading) {
    return (
      <div className="space-y-4">
        {Array.from({ length: 5 }).map((_, i) => (
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
          <Settings className="size-5 text-muted-foreground" />
          {t("config.visual_title")}
        </CardTitle>
        <CardDescription>{t("config.visual_desc")}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        {SAFE_FIELDS.map((field) => {
          const type = fieldType(field);

          if (type === "boolean") {
            return (
              <div key={field} className="flex items-center justify-between max-w-lg">
                <Label htmlFor={field}>{t(`config.field_${field.replace(/-/g, "_")}`)}</Label>
                <Switch
                  id={field}
                  checked={!!draft[field]}
                  onCheckedChange={(checked) => updateField(field, checked)}
                />
              </div>
            );
          }

          if (type === "select" && field === "routing-strategy") {
            return (
              <div key={field} className="grid max-w-lg gap-2">
                <Label htmlFor={field}>{t(`config.field_${field.replace(/-/g, "_")}`)}</Label>
                <Select
                  value={(draft[field] as string) ?? ""}
                  onValueChange={(v) => updateField(field, v)}
                >
                  <SelectTrigger id={field}>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {ROUTING_STRATEGIES.map((s) => (
                      <SelectItem key={s} value={s}>
                        {t(`routing_strategy.option_${s.replace(/-/g, "_")}`)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            );
          }

          // String input (e.g. proxy-url).
          return (
            <div key={field} className="grid max-w-lg gap-2">
              <Label htmlFor={field}>{t(`config.field_${field.replace(/-/g, "_")}`)}</Label>
              <Input
                id={field}
                value={(draft[field] as string) ?? ""}
                placeholder={t("common.not_set")}
                onChange={(e) => updateField(field, e.target.value || undefined)}
              />
            </div>
          );
        })}

        <Button onClick={save} loading={mutation.isPending} disabled={!dirty}>
          {t("common.save")}
        </Button>
      </CardContent>
    </Card>
  );
}
