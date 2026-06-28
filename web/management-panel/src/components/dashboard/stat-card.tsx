import * as React from "react";
import { type LucideIcon } from "lucide-react";
import { Card, Skeleton } from "@/components/ui";
import { cn } from "@/lib/utils";

export interface StatCardProps {
  label: string;
  value: React.ReactNode;
  icon: LucideIcon;
  hint?: string;
  loading?: boolean;
  tone?: "default" | "success" | "warning" | "danger" | "info";
  className?: string;
}

const toneAccent: Record<NonNullable<StatCardProps["tone"]>, string> = {
  default: "text-brand-500 bg-brand-500/10",
  success: "text-success bg-success/10",
  warning: "text-warning bg-warning/10",
  danger: "text-danger bg-danger/10",
  info: "text-info bg-info/10",
};

/** Compact KPI tile for the Overview dashboard. */
export function StatCard({
  label,
  value,
  icon: Icon,
  hint,
  loading = false,
  tone = "default",
  className,
}: StatCardProps) {
  return (
    <Card className={cn("p-5", className)}>
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 space-y-1">
          <p className="truncate text-xs font-medium uppercase tracking-wide text-muted-foreground">
            {label}
          </p>
          {loading ? (
            <Skeleton className="mt-1 h-7 w-24" />
          ) : (
            <p className="text-2xl font-semibold tracking-tight text-foreground tabular-nums">
              {value}
            </p>
          )}
          {hint ? <p className="truncate text-xs text-muted-foreground">{hint}</p> : null}
        </div>
        <span className={cn("flex size-9 shrink-0 items-center justify-center rounded-lg", toneAccent[tone])}>
          <Icon className="size-5" />
        </span>
      </div>
    </Card>
  );
}
