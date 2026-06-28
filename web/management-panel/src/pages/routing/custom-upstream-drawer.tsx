import * as React from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Plus, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { ApiError } from "@/lib/api";
import {
  type CustomUpstream,
  type UpstreamModel,
  useCreateCustomUpstreamMutation,
  useUpdateCustomUpstreamMutation,
} from "@/lib/api/custom-upstreams";
import {
  Button,
  Drawer,
  DrawerBody,
  DrawerContent,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
  DrawerDescription,
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  Input,
  Switch,
  toast,
} from "@/components/ui";
import { KeyValueField, type KeyValuePair } from "@/components/forms/key-value-field";
import { pairsToRecord, recordToPairs } from "@/components/forms/key-value-utils";
import {
  ModelMappingField,
  type ModelMappingRow,
} from "@/components/forms/model-mapping-field";

/** Zod schema for the upstream editor. Keys/models/headers are managed as
 * local state outside the resolver since they are dynamic arrays. */
const upstreamSchema = z.object({
  name: z.string().trim().min(1, { message: "errors.name_required" }),
  baseUrl: z
    .string()
    .trim()
    .min(1, { message: "errors.base_url_required" })
    .url({ message: "errors.base_url_invalid" }),
  prefix: z.string().trim().optional(),
  priority: z.coerce.number().int().optional(),
  disabled: z.boolean(),
});

type UpstreamFormValues = z.input<typeof upstreamSchema>;

export interface CustomUpstreamDrawerProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** The upstream being edited, or null when creating a new one. */
  upstream: CustomUpstream | null;
  /** Current list, used to append on create and to detect name collisions. */
  existing: CustomUpstream[];
}

function modelsToRows(models: UpstreamModel[]): ModelMappingRow[] {
  return models.map((m) => ({
    name: m.name,
    alias: m.alias,
    forceMapping: m["force-mapping"] ?? false,
    image: m.image ?? false,
  }));
}

/**
 * Create/edit drawer for a custom upstream. On create it appends to the list
 * and PUTs the whole collection; on edit it PATCHes by the original name so
 * unrelated upstreams are untouched. The opaque per-model `thinking` descriptor
 * is preserved across edits to avoid silent data loss.
 */
export function CustomUpstreamDrawer({
  open,
  onOpenChange,
  upstream,
  existing,
}: CustomUpstreamDrawerProps) {
  const { t } = useTranslation();
  const isEdit = upstream !== null;

  const createMutation = useCreateCustomUpstreamMutation();
  const updateMutation = useUpdateCustomUpstreamMutation();
  const pending = createMutation.isPending || updateMutation.isPending;

  const [apiKeys, setApiKeys] = React.useState<{ key: string; proxy: string }[]>([]);
  const [models, setModels] = React.useState<ModelMappingRow[]>([]);
  const [headers, setHeaders] = React.useState<KeyValuePair[]>([]);

  const form = useForm<UpstreamFormValues>({
    resolver: zodResolver(upstreamSchema),
    defaultValues: { name: "", baseUrl: "", prefix: "", priority: 0, disabled: false },
  });

  // Reset all controlled state whenever the drawer opens for a new target.
  React.useEffect(() => {
    if (!open) return;
    form.reset({
      name: upstream?.name ?? "",
      baseUrl: upstream?.["base-url"] ?? "",
      prefix: upstream?.prefix ?? "",
      priority: upstream?.priority ?? 0,
      disabled: upstream?.disabled ?? false,
    });
    setApiKeys(
      (upstream?.["api-key-entries"] ?? []).map((entry) => ({
        key: entry["api-key"],
        proxy: entry["proxy-url"] ?? "",
      })),
    );
    setModels(modelsToRows(upstream?.models ?? []));
    setHeaders(recordToPairs(upstream?.headers));
    // form is stable; depending on it would re-run on every keystroke.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, upstream]);

  const onSubmit = form.handleSubmit((values) => {
    const trimmedName = values.name.trim();
    // Guard against name collisions (create, or rename onto another entry).
    const collision = existing.some(
      (u) => u.name === trimmedName && u.name !== upstream?.name,
    );
    if (collision) {
      form.setError("name", { message: "errors.name_duplicate" });
      return;
    }

    const builtModels: UpstreamModel[] = models
      .filter((m) => m.name.trim() || m.alias.trim())
      .map((m, index) => {
        // Preserve the opaque thinking descriptor from the original model, if any.
        const original = upstream?.models?.[index];
        return {
          name: m.name.trim(),
          alias: m.alias.trim(),
          "force-mapping": m.forceMapping,
          ...(m.image ? { image: true } : {}),
          ...(original?.thinking ? { thinking: original.thinking } : {}),
        };
      });

    const keyEntries = apiKeys
      .filter((entry) => entry.key.trim())
      .map((entry) => ({
        "api-key": entry.key.trim(),
        ...(entry.proxy.trim() ? { "proxy-url": entry.proxy.trim() } : {}),
      }));

    const headerRecord = pairsToRecord(headers);
    const priority = Number(values.priority ?? 0);

    const payload: CustomUpstream = {
      name: trimmedName,
      "base-url": values.baseUrl.trim(),
      models: builtModels,
      disabled: values.disabled,
      ...(values.prefix?.trim() ? { prefix: values.prefix.trim() } : {}),
      ...(Number.isFinite(priority) && priority !== 0 ? { priority } : {}),
      ...(keyEntries.length ? { "api-key-entries": keyEntries } : {}),
      ...(Object.keys(headerRecord).length ? { headers: headerRecord } : {}),
      ...(upstream?.["disable-cooling"] ? { "disable-cooling": true } : {}),
    };

    const handleError = (error: unknown) => {
      const message =
        error instanceof ApiError ? error.message : t("common.unknown_error");
      toast.error(t("custom_upstreams.save_failed"), message);
    };

    if (isEdit && upstream) {
      updateMutation.mutate(
        { name: upstream.name, value: payload },
        {
          onSuccess: () => {
            toast.success(t("custom_upstreams.saved"));
            onOpenChange(false);
          },
          onError: handleError,
        },
      );
    } else {
      createMutation.mutate(
        { existing, upstream: payload },
        {
          onSuccess: () => {
            toast.success(t("custom_upstreams.created"));
            onOpenChange(false);
          },
          onError: handleError,
        },
      );
    }
  });

  const updateApiKey = (index: number, patch: Partial<{ key: string; proxy: string }>) => {
    setApiKeys((prev) => {
      const copy = [...prev];
      copy[index] = { ...copy[index]!, ...patch };
      return copy;
    });
  };

  return (
    <Drawer open={open} onOpenChange={onOpenChange}>
      <DrawerContent className="max-w-xl">
        <DrawerHeader>
          <DrawerTitle>
            {isEdit ? t("custom_upstreams.edit_title") : t("custom_upstreams.create_title")}
          </DrawerTitle>
          <DrawerDescription>{t("custom_upstreams.drawer_desc")}</DrawerDescription>
        </DrawerHeader>

        <Form {...form}>
          <form onSubmit={onSubmit} className="flex min-h-0 flex-1 flex-col">
            <DrawerBody className="space-y-6">
              {/* Identity */}
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <FormField
                  control={form.control}
                  name="name"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t("custom_upstreams.field_name")}</FormLabel>
                      <FormControl>
                        <Input
                          {...field}
                          placeholder={t("custom_upstreams.field_name_placeholder")}
                        />
                      </FormControl>
                      <FormMessage>
                        {form.formState.errors.name?.message
                          ? t(form.formState.errors.name.message)
                          : null}
                      </FormMessage>
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="prefix"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t("custom_upstreams.field_prefix")}</FormLabel>
                      <FormControl>
                        <Input
                          {...field}
                          placeholder={t("custom_upstreams.field_prefix_placeholder")}
                        />
                      </FormControl>
                      <FormDescription>{t("custom_upstreams.field_prefix_desc")}</FormDescription>
                    </FormItem>
                  )}
                />
              </div>

              <FormField
                control={form.control}
                name="baseUrl"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t("custom_upstreams.field_base_url")}</FormLabel>
                    <FormControl>
                      <Input {...field} placeholder="https://api.example.com/v1" className="font-mono" />
                    </FormControl>
                    <FormMessage>
                      {form.formState.errors.baseUrl?.message
                        ? t(form.formState.errors.baseUrl.message)
                        : null}
                    </FormMessage>
                  </FormItem>
                )}
              />

              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <FormField
                  control={form.control}
                  name="priority"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t("custom_upstreams.field_priority")}</FormLabel>
                      <FormControl>
                        <Input
                          type="number"
                          value={field.value ?? 0}
                          onChange={(e) => field.onChange(e.target.value)}
                          onBlur={field.onBlur}
                          name={field.name}
                          ref={field.ref}
                        />
                      </FormControl>
                      <FormDescription>{t("custom_upstreams.field_priority_desc")}</FormDescription>
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="disabled"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t("custom_upstreams.field_disabled")}</FormLabel>
                      <FormControl>
                        <div className="flex h-9 items-center">
                          <Switch checked={field.value} onCheckedChange={field.onChange} />
                        </div>
                      </FormControl>
                      <FormDescription>{t("custom_upstreams.field_disabled_desc")}</FormDescription>
                    </FormItem>
                  )}
                />
              </div>

              {/* API key entries */}
              <div className="space-y-2">
                <p className="text-sm font-medium text-foreground">
                  {t("custom_upstreams.section_keys")}
                </p>
                <div className="space-y-2">
                  {apiKeys.map((entry, index) => (
                    <div key={index} className="flex items-center gap-2">
                      <Input
                        value={entry.key}
                        placeholder={t("custom_upstreams.api_key_placeholder")}
                        onChange={(e) => updateApiKey(index, { key: e.target.value })}
                        className="font-mono"
                      />
                      <Input
                        value={entry.proxy}
                        placeholder={t("custom_upstreams.proxy_url_placeholder")}
                        onChange={(e) => updateApiKey(index, { proxy: e.target.value })}
                        className="font-mono"
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon-sm"
                        onClick={() => setApiKeys((prev) => prev.filter((_, i) => i !== index))}
                        aria-label={t("common.remove")}
                      >
                        <X className="size-4" />
                      </Button>
                    </div>
                  ))}
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => setApiKeys((prev) => [...prev, { key: "", proxy: "" }])}
                  >
                    <Plus className="size-4" />
                    {t("custom_upstreams.add_key")}
                  </Button>
                </div>
              </div>

              {/* Model mappings */}
              <div className="space-y-2">
                <p className="text-sm font-medium text-foreground">
                  {t("custom_upstreams.section_models")}
                </p>
                <ModelMappingField value={models} onChange={setModels} showImage />
              </div>

              {/* Headers */}
              <div className="space-y-2">
                <p className="text-sm font-medium text-foreground">
                  {t("custom_upstreams.section_headers")}
                </p>
                <KeyValueField value={headers} onChange={setHeaders} />
              </div>
            </DrawerBody>

            <DrawerFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => onOpenChange(false)}
                disabled={pending}
              >
                {t("common.cancel")}
              </Button>
              <Button type="submit" loading={pending}>
                {t("common.save")}
              </Button>
            </DrawerFooter>
          </form>
        </Form>
      </DrawerContent>
    </Drawer>
  );
}
