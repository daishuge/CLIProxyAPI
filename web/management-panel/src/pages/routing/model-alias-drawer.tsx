import * as React from "react";
import { Plus, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { ApiError } from "@/lib/api";
import {
  type ModelAlias,
  type ModelAliasMap,
  useUpdateModelAliasesMutation,
} from "@/lib/api/model-aliases";
import { MODEL_CHANNELS } from "@/lib/api/models";
import {
  Button,
  Checkbox,
  Drawer,
  DrawerBody,
  DrawerContent,
  DrawerDescription,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
  Input,
  Label,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  toast,
} from "@/components/ui";

interface AliasRow {
  name: string;
  alias: string;
  forceMapping: boolean;
}

export interface ModelAliasDrawerProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Channel being edited, or null when adding a new channel. */
  channel: string | null;
  aliasMap: ModelAliasMap;
}

/** Create/edit the alias list for a single channel. */
export function ModelAliasDrawer({ open, onOpenChange, channel, aliasMap }: ModelAliasDrawerProps) {
  const { t } = useTranslation();
  const isEdit = channel !== null;
  const mutation = useUpdateModelAliasesMutation();

  const [channelName, setChannelName] = React.useState<string>("");
  const [rows, setRows] = React.useState<AliasRow[]>([]);

  React.useEffect(() => {
    if (!open) return;
    setChannelName(channel ?? MODEL_CHANNELS[0]);
    setRows(
      (channel ? (aliasMap[channel] ?? []) : []).map((a) => ({
        name: a.name,
        alias: a.alias,
        forceMapping: a["force-mapping"] ?? false,
      })),
    );
  }, [open, channel, aliasMap]);

  const update = (index: number, patch: Partial<AliasRow>) =>
    setRows((prev) => prev.map((r, i) => (i === index ? { ...r, ...patch } : r)));
  const remove = (index: number) => setRows((prev) => prev.filter((_, i) => i !== index));
  const add = () => setRows((prev) => [...prev, { name: "", alias: "", forceMapping: false }]);

  const save = () => {
    const targetChannel = channelName.trim().toLowerCase();
    if (!targetChannel) return;
    const aliases: ModelAlias[] = rows
      .filter((r) => r.name.trim() && r.alias.trim())
      .map((r) => ({
        name: r.name.trim(),
        alias: r.alias.trim(),
        ...(r.forceMapping ? { "force-mapping": true } : {}),
      }));

    mutation.mutate(
      { channel: targetChannel, aliases },
      {
        onSuccess: () => {
          toast.success(t("model_aliases.saved"));
          onOpenChange(false);
        },
        onError: (error) => {
          const message = error instanceof ApiError ? error.message : t("common.unknown_error");
          toast.error(t("model_aliases.save_failed"), message);
        },
      },
    );
  };

  return (
    <Drawer open={open} onOpenChange={onOpenChange}>
      <DrawerContent>
        <DrawerHeader>
          <DrawerTitle>
            {isEdit ? t("model_aliases.edit_title") : t("model_aliases.create_title")}
          </DrawerTitle>
          <DrawerDescription>{t("model_aliases.drawer_desc")}</DrawerDescription>
        </DrawerHeader>
        <DrawerBody className="space-y-5">
          <div className="space-y-2">
            <Label htmlFor="alias-channel">{t("model_aliases.field_channel")}</Label>
            {isEdit ? (
              <Input id="alias-channel" value={channelName} disabled className="font-mono" />
            ) : (
              <Select value={channelName} onValueChange={setChannelName}>
                <SelectTrigger id="alias-channel">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {MODEL_CHANNELS.map((c) => (
                    <SelectItem key={c} value={c} className="capitalize">
                      {c}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
          </div>

          <div className="space-y-2">
            <p className="text-sm font-medium text-foreground">{t("model_aliases.section_aliases")}</p>
            {rows.map((row, index) => (
              <div key={index} className="rounded-md border border-border bg-surface-sunken/40 p-3">
                <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
                  <div className="space-y-1">
                    <Label htmlFor={`alias-name-${index}`} className="text-xs">
                      {t("model_aliases.field_alias")}
                    </Label>
                    <Input
                      id={`alias-name-${index}`}
                      value={row.alias}
                      placeholder={t("model_aliases.alias_placeholder")}
                      onChange={(e) => update(index, { alias: e.target.value })}
                      className="font-mono"
                    />
                  </div>
                  <div className="space-y-1">
                    <Label htmlFor={`alias-target-${index}`} className="text-xs">
                      {t("model_aliases.field_target")}
                    </Label>
                    <Input
                      id={`alias-target-${index}`}
                      value={row.name}
                      placeholder={t("model_aliases.target_placeholder")}
                      onChange={(e) => update(index, { name: e.target.value })}
                      className="font-mono"
                    />
                  </div>
                </div>
                <div className="mt-2 flex items-center justify-between">
                  <label className="flex items-center gap-2 text-xs text-muted-foreground">
                    <Checkbox
                      checked={row.forceMapping}
                      onCheckedChange={(checked) => update(index, { forceMapping: checked === true })}
                    />
                    {t("models_field.force_mapping")}
                  </label>
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-sm"
                    onClick={() => remove(index)}
                    aria-label={t("common.remove")}
                  >
                    <X className="size-4" />
                  </Button>
                </div>
              </div>
            ))}
            <Button type="button" variant="outline" size="sm" onClick={add}>
              <Plus className="size-4" />
              {t("model_aliases.add_alias")}
            </Button>
          </div>
        </DrawerBody>
        <DrawerFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={mutation.isPending}>
            {t("common.cancel")}
          </Button>
          <Button onClick={save} loading={mutation.isPending}>
            {t("common.save")}
          </Button>
        </DrawerFooter>
      </DrawerContent>
    </Drawer>
  );
}
