import * as React from "react";
import { useTranslation } from "react-i18next";
import { ApiError } from "@/lib/api";
import {
  type ProviderKey,
  type ProviderKeyType,
  PROVIDERS_WITH_MODELS,
  useCreateProviderKeyMutation,
  useUpdateProviderKeyMutation,
} from "@/lib/api/provider-keys";
import {
  Button,
  Drawer,
  DrawerBody,
  DrawerContent,
  DrawerDescription,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
  Input,
  Label,
  toast,
} from "@/components/ui";
import { KeyValueField, type KeyValuePair } from "@/components/forms/key-value-field";
import { pairsToRecord, recordToPairs } from "@/components/forms/key-value-utils";
import { StringListField } from "@/components/forms/string-list-field";
import {
  ModelMappingField,
  type ModelMappingRow,
} from "@/components/forms/model-mapping-field";

export interface ProviderKeyDrawerProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  provider: ProviderKeyType;
  /** The key being edited, or null when creating. */
  providerKey: ProviderKey | null;
  /** Current list, used to append on create. */
  existing: ProviderKey[];
}

/** Create/edit a provider API key. Model mappings only render for claude/codex. */
export function ProviderKeyDrawer({
  open,
  onOpenChange,
  provider,
  providerKey,
  existing,
}: ProviderKeyDrawerProps) {
  const { t } = useTranslation();
  const isEdit = providerKey !== null;
  const supportsModels = PROVIDERS_WITH_MODELS.has(provider);

  const createMutation = useCreateProviderKeyMutation(provider);
  const updateMutation = useUpdateProviderKeyMutation(provider);
  const pending = createMutation.isPending || updateMutation.isPending;

  const [apiKey, setApiKey] = React.useState("");
  const [baseUrl, setBaseUrl] = React.useState("");
  const [proxyUrl, setProxyUrl] = React.useState("");
  const [prefix, setPrefix] = React.useState("");
  const [priority, setPriority] = React.useState("0");
  const [excluded, setExcluded] = React.useState<string[]>([]);
  const [headers, setHeaders] = React.useState<KeyValuePair[]>([]);
  const [models, setModels] = React.useState<ModelMappingRow[]>([]);
  const [apiKeyError, setApiKeyError] = React.useState(false);

  React.useEffect(() => {
    if (!open) return;
    setApiKey(providerKey?.["api-key"] ?? "");
    setBaseUrl(providerKey?.["base-url"] ?? "");
    setProxyUrl(providerKey?.["proxy-url"] ?? "");
    setPrefix(providerKey?.prefix ?? "");
    setPriority(String(providerKey?.priority ?? 0));
    setExcluded(providerKey?.["excluded-models"] ?? []);
    setHeaders(recordToPairs(providerKey?.headers));
    setModels(
      (providerKey?.models ?? []).map((m) => ({
        name: m.name,
        alias: m.alias,
        forceMapping: m["force-mapping"] ?? false,
      })),
    );
    setApiKeyError(false);
  }, [open, providerKey]);

  const save = () => {
    const trimmedKey = apiKey.trim();
    if (!trimmedKey) {
      setApiKeyError(true);
      return;
    }

    const headerRecord = pairsToRecord(headers);
    const excludedClean = excluded.map((e) => e.trim()).filter(Boolean);
    const priorityNum = Number(priority);
    const builtModels = models
      .filter((m) => m.name.trim() || m.alias.trim())
      .map((m) => ({
        name: m.name.trim(),
        alias: m.alias.trim(),
        "force-mapping": m.forceMapping,
      }));

    const payload: ProviderKey = {
      "api-key": trimmedKey,
      ...(baseUrl.trim() ? { "base-url": baseUrl.trim() } : {}),
      ...(proxyUrl.trim() ? { "proxy-url": proxyUrl.trim() } : {}),
      ...(prefix.trim() ? { prefix: prefix.trim() } : {}),
      ...(Number.isFinite(priorityNum) && priorityNum !== 0 ? { priority: priorityNum } : {}),
      ...(excludedClean.length ? { "excluded-models": excludedClean } : {}),
      ...(Object.keys(headerRecord).length ? { headers: headerRecord } : {}),
      ...(supportsModels && builtModels.length ? { models: builtModels } : {}),
    };

    const handleError = (error: unknown) => {
      const message = error instanceof ApiError ? error.message : t("common.unknown_error");
      toast.error(t("provider_keys.save_failed"), message);
    };

    if (isEdit && providerKey) {
      updateMutation.mutate(
        { match: providerKey["api-key"], value: payload },
        {
          onSuccess: () => {
            toast.success(t("provider_keys.saved"));
            onOpenChange(false);
          },
          onError: handleError,
        },
      );
    } else {
      createMutation.mutate(
        { existing, key: payload },
        {
          onSuccess: () => {
            toast.success(t("provider_keys.created"));
            onOpenChange(false);
          },
          onError: handleError,
        },
      );
    }
  };

  return (
    <Drawer open={open} onOpenChange={onOpenChange}>
      <DrawerContent className="max-w-lg">
        <DrawerHeader>
          <DrawerTitle>
            {isEdit ? t("provider_keys.edit_title") : t("provider_keys.create_title")}
          </DrawerTitle>
          <DrawerDescription>
            {t("provider_keys.drawer_desc", {
              provider: t(`provider_keys.provider_${provider}`),
            })}
          </DrawerDescription>
        </DrawerHeader>
        <DrawerBody className="space-y-5">
          <div className="space-y-2">
            <Label htmlFor="pk-api-key">{t("provider_keys.field_api_key")}</Label>
            <Input
              id="pk-api-key"
              value={apiKey}
              invalid={apiKeyError}
              onChange={(e) => {
                setApiKey(e.target.value);
                if (apiKeyError) setApiKeyError(false);
              }}
              placeholder={t("provider_keys.api_key_placeholder")}
              className="font-mono"
            />
            {apiKeyError ? (
              <p className="text-xs font-medium text-danger">{t("errors.api_key_required")}</p>
            ) : null}
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="pk-base-url">{t("provider_keys.field_base_url")}</Label>
              <Input
                id="pk-base-url"
                value={baseUrl}
                onChange={(e) => setBaseUrl(e.target.value)}
                placeholder={t("provider_keys.base_url_placeholder")}
                className="font-mono"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="pk-proxy-url">{t("provider_keys.field_proxy_url")}</Label>
              <Input
                id="pk-proxy-url"
                value={proxyUrl}
                onChange={(e) => setProxyUrl(e.target.value)}
                placeholder={t("provider_keys.proxy_url_placeholder")}
                className="font-mono"
              />
            </div>
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="pk-prefix">{t("provider_keys.field_prefix")}</Label>
              <Input
                id="pk-prefix"
                value={prefix}
                onChange={(e) => setPrefix(e.target.value)}
                placeholder={t("provider_keys.prefix_placeholder")}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="pk-priority">{t("provider_keys.field_priority")}</Label>
              <Input
                id="pk-priority"
                type="number"
                value={priority}
                onChange={(e) => setPriority(e.target.value)}
              />
            </div>
          </div>

          {supportsModels ? (
            <div className="space-y-2">
              <p className="text-sm font-medium text-foreground">
                {t("provider_keys.section_models")}
              </p>
              <ModelMappingField value={models} onChange={setModels} />
            </div>
          ) : null}

          <div className="space-y-2">
            <Label>{t("provider_keys.section_excluded")}</Label>
            <StringListField
              value={excluded}
              onChange={setExcluded}
              placeholder={t("provider_keys.excluded_placeholder")}
              addLabel={t("provider_keys.add_excluded")}
            />
          </div>

          <div className="space-y-2">
            <p className="text-sm font-medium text-foreground">{t("provider_keys.section_headers")}</p>
            <KeyValueField value={headers} onChange={setHeaders} />
          </div>
        </DrawerBody>
        <DrawerFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={pending}>
            {t("common.cancel")}
          </Button>
          <Button onClick={save} loading={pending}>
            {t("common.save")}
          </Button>
        </DrawerFooter>
      </DrawerContent>
    </Drawer>
  );
}
