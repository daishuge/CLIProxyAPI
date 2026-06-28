import { useTranslation } from "react-i18next";
import { APP_VERSION } from "@/lib/version";
import { useLatestVersionQuery } from "@/lib/hooks";
import { Badge, Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui";

/** Displays the running build and flags when a newer release exists. */
export function VersionPill() {
  const { t } = useTranslation();
  const { data } = useLatestVersionQuery();
  const latest = data?.["latest-version"];
  const normalizedLatest = latest?.replace(/^v/, "");
  const hasUpdate = !!normalizedLatest && normalizedLatest !== APP_VERSION;

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Badge variant={hasUpdate ? "warning" : "outline"} className="font-mono">
          v{APP_VERSION}
        </Badge>
      </TooltipTrigger>
      <TooltipContent>
        {hasUpdate
          ? t("header.latest_version", { version: latest })
          : t("header.up_to_date")}
      </TooltipContent>
    </Tooltip>
  );
}
