import { Plus, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button, Input } from "@/components/ui";

export interface StringListFieldProps {
  value: string[];
  onChange: (next: string[]) => void;
  placeholder?: string;
  addLabel?: string;
  /** Optional id for the first input, to wire an external label. */
  inputId?: string;
}

/**
 * Editable list of free-text strings (e.g. excluded model ids, reasoning
 * levels). Renders one input per entry with add/remove controls.
 */
export function StringListField({
  value,
  onChange,
  placeholder,
  addLabel,
  inputId,
}: StringListFieldProps) {
  const { t } = useTranslation();

  const update = (index: number, next: string) => {
    const copy = [...value];
    copy[index] = next;
    onChange(copy);
  };
  const remove = (index: number) => onChange(value.filter((_, i) => i !== index));
  const add = () => onChange([...value, ""]);

  return (
    <div className="space-y-2">
      {value.map((entry, index) => (
        <div key={index} className="flex items-center gap-2">
          <Input
            id={index === 0 ? inputId : undefined}
            value={entry}
            placeholder={placeholder}
            onChange={(e) => update(index, e.target.value)}
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
