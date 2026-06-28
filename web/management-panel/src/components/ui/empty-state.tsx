import * as React from "react";
import { type LucideIcon } from "lucide-react";
import { cn } from "@/lib/utils";

export interface EmptyStateProps extends React.HTMLAttributes<HTMLDivElement> {
  icon?: LucideIcon;
  title: string;
  description?: string;
  action?: React.ReactNode;
  tone?: "neutral" | "danger";
}

/**
 * Unified empty/error surface. `tone="danger"` doubles as the error state so
 * data views render the same component for both "no data" and "load failed".
 */
export function EmptyState({
  icon: Icon,
  title,
  description,
  action,
  tone = "neutral",
  className,
  ...props
}: EmptyStateProps) {
  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed px-6 py-12 text-center",
        tone === "danger" ? "border-danger/40 bg-danger/[0.04]" : "border-border bg-surface-sunken/40",
        className,
      )}
      {...props}
    >
      {Icon ? (
        <span
          className={cn(
            "flex size-11 items-center justify-center rounded-full",
            tone === "danger" ? "bg-danger/10 text-danger" : "bg-muted text-muted-foreground",
          )}
        >
          <Icon className="size-5" />
        </span>
      ) : null}
      <div className="space-y-1">
        <p className="text-sm font-semibold text-foreground">{title}</p>
        {description ? (
          <p className="mx-auto max-w-sm text-sm text-muted-foreground">{description}</p>
        ) : null}
      </div>
      {action ? <div className="mt-1">{action}</div> : null}
    </div>
  );
}
