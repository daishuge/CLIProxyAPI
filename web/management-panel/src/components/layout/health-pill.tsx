import { useTranslation } from "react-i18next";
import { useHealthQuery } from "@/lib/hooks";
import { cn } from "@/lib/utils";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui";

/** Compact backend health indicator shown in the topbar. */
export function HealthPill() {
  const { t } = useTranslation();
  const { data, isError, isLoading, isFetching } = useHealthQuery();

  const healthy = data?.status === "ok" && !isError;
  const state: "healthy" | "offline" | "checking" = isLoading
    ? "checking"
    : healthy
      ? "healthy"
      : "offline";

  const dotColor =
    state === "healthy"
      ? "bg-success"
      : state === "offline"
        ? "bg-danger"
        : "bg-warning";

  const label =
    state === "healthy"
      ? t("header.health_healthy")
      : state === "offline"
        ? t("header.health_offline")
        : t("header.health_checking");

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span
          className={cn(
            "inline-flex items-center gap-1.5 rounded-full border border-border bg-surface px-2.5 py-1 text-xs font-medium",
            isFetching && "opacity-90",
          )}
        >
          <span className="relative flex size-2">
            {state === "healthy" ? (
              <span
                className={cn(
                  "absolute inline-flex size-full rounded-full opacity-60",
                  "animate-[ppap-pulse-ring_1.8s_cubic-bezier(0.4,0,0.6,1)_infinite]",
                  dotColor,
                )}
              />
            ) : null}
            <span className={cn("relative inline-flex size-2 rounded-full", dotColor)} />
          </span>
          {label}
        </span>
      </TooltipTrigger>
      <TooltipContent>{t("header.health_healthy") + " · /healthz"}</TooltipContent>
    </Tooltip>
  );
}
