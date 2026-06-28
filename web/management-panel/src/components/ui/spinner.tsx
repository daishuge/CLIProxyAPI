import { Loader2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { cn } from "@/lib/utils";

export interface SpinnerProps {
  className?: string;
  /** Diameter token; maps to a tailwind size utility. */
  size?: "sm" | "md" | "lg";
  label?: string;
}

const sizeMap = {
  sm: "size-4",
  md: "size-6",
  lg: "size-8",
} as const;

export function Spinner({ className, size = "md", label }: SpinnerProps) {
  const { t } = useTranslation();
  return (
    <span role="status" aria-live="polite" className={cn("inline-flex items-center gap-2", className)}>
      <Loader2
        className={cn("animate-[ppap-spin_0.7s_linear_infinite] text-muted-foreground", sizeMap[size])}
      />
      {label ? <span className="text-sm text-muted-foreground">{label}</span> : null}
      <span className="sr-only">{label ?? t("common.loading")}</span>
    </span>
  );
}
