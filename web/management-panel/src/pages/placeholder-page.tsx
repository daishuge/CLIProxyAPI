import { Construction } from "lucide-react";
import { useTranslation } from "react-i18next";
import { PageHeader } from "@/components/layout/page-header";
import { EmptyState } from "@/components/ui";

/** Generic "coming soon" page used for IA routes not yet implemented. */
export function PlaceholderPage({ titleKey }: { titleKey: string }) {
  const { t } = useTranslation();
  return (
    <div className="space-y-6">
      <PageHeader title={t(titleKey)} />
      <EmptyState
        icon={Construction}
        title={t("placeholder.coming_soon_title")}
        description={t("placeholder.coming_soon_desc")}
      />
    </div>
  );
}
