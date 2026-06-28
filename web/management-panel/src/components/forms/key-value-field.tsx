import { Plus, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button, Input } from "@/components/ui";

/** A header key/value pair held during editing (kept as an array to allow
 * empty/duplicate keys while typing; serialized to a record on submit). */
export interface KeyValuePair {
  key: string;
  value: string;
}

export interface KeyValueFieldProps {
  value: KeyValuePair[];
  onChange: (next: KeyValuePair[]) => void;
  keyPlaceholder?: string;
  valuePlaceholder?: string;
  addLabel?: string;
}

/** Editable list of key/value pairs, used for arbitrary HTTP headers. */
export function KeyValueField({
  value,
  onChange,
  keyPlaceholder,
  valuePlaceholder,
  addLabel,
}: KeyValueFieldProps) {
  const { t } = useTranslation();

  const update = (index: number, patch: Partial<KeyValuePair>) => {
    const copy = [...value];
    copy[index] = { ...copy[index]!, ...patch };
    onChange(copy);
  };
  const remove = (index: number) => onChange(value.filter((_, i) => i !== index));
  const add = () => onChange([...value, { key: "", value: "" }]);

  return (
    <div className="space-y-2">
      {value.map((pair, index) => (
        <div key={index} className="flex items-center gap-2">
          <Input
            value={pair.key}
            placeholder={keyPlaceholder ?? "Header"}
            onChange={(e) => update(index, { key: e.target.value })}
            className="font-mono"
          />
          <Input
            value={pair.value}
            placeholder={valuePlaceholder ?? "Value"}
            onChange={(e) => update(index, { value: e.target.value })}
            className="font-mono"
          />
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
      ))}
      <Button type="button" variant="outline" size="sm" onClick={add}>
        <Plus className="size-4" />
        {addLabel ?? t("common.add")}
      </Button>
    </div>
  );
}
