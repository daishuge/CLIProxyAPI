import { Plus, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button, Checkbox, Input, Label } from "@/components/ui";

/** One model mapping row during editing. `image` is only meaningful for
 * upstream models that support image generation. */
export interface ModelMappingRow {
  name: string;
  alias: string;
  forceMapping: boolean;
  image?: boolean;
}

export interface ModelMappingFieldProps {
  value: ModelMappingRow[];
  onChange: (next: ModelMappingRow[]) => void;
  /** Show the per-row "image" capability checkbox (custom upstreams only). */
  showImage?: boolean;
}

/**
 * Editable table of upstream model -> client alias mappings with per-row
 * force-mapping (and optional image) flags.
 */
export function ModelMappingField({ value, onChange, showImage = false }: ModelMappingFieldProps) {
  const { t } = useTranslation();

  const update = (index: number, patch: Partial<ModelMappingRow>) => {
    const copy = [...value];
    copy[index] = { ...copy[index]!, ...patch };
    onChange(copy);
  };
  const remove = (index: number) => onChange(value.filter((_, i) => i !== index));
  const add = () => onChange([...value, { name: "", alias: "", forceMapping: false }]);

  return (
    <div className="space-y-3">
      {value.length > 0 ? (
        <div className="space-y-2">
          {value.map((row, index) => (
            <div
              key={index}
              className="rounded-md border border-border bg-surface-sunken/40 p-3"
            >
              <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
                <div className="space-y-1">
                  <Label htmlFor={`model-name-${index}`} className="text-xs">
                    {t("models_field.upstream_name")}
                  </Label>
                  <Input
                    id={`model-name-${index}`}
                    value={row.name}
                    placeholder={t("models_field.upstream_placeholder")}
                    onChange={(e) => update(index, { name: e.target.value })}
                    className="font-mono"
                  />
                </div>
                <div className="space-y-1">
                  <Label htmlFor={`model-alias-${index}`} className="text-xs">
                    {t("models_field.alias")}
                  </Label>
                  <Input
                    id={`model-alias-${index}`}
                    value={row.alias}
                    placeholder={t("models_field.alias_placeholder")}
                    onChange={(e) => update(index, { alias: e.target.value })}
                    className="font-mono"
                  />
                </div>
              </div>
              <div className="mt-3 flex items-center justify-between">
                <div className="flex items-center gap-4">
                  <label className="flex items-center gap-2 text-xs text-muted-foreground">
                    <Checkbox
                      checked={row.forceMapping}
                      onCheckedChange={(checked) => update(index, { forceMapping: checked === true })}
                    />
                    {t("models_field.force_mapping")}
                  </label>
                  {showImage ? (
                    <label className="flex items-center gap-2 text-xs text-muted-foreground">
                      <Checkbox
                        checked={row.image ?? false}
                        onCheckedChange={(checked) => update(index, { image: checked === true })}
                      />
                      {t("models_field.image")}
                    </label>
                  ) : null}
                </div>
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
        </div>
      ) : null}
      <Button type="button" variant="outline" size="sm" onClick={add}>
        <Plus className="size-4" />
        {t("models_field.add_model")}
      </Button>
    </div>
  );
}
