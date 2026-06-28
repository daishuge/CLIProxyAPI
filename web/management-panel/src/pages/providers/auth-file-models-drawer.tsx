import { useTranslation } from "react-i18next";
import { useAuthFileModelsQuery } from "@/lib/api/auth-files";
import {
  Badge,
  Drawer,
  DrawerBody,
  DrawerContent,
  DrawerDescription,
  DrawerHeader,
  DrawerTitle,
  EmptyState,
  Skeleton,
} from "@/components/ui";

export interface AuthFileModelsDrawerProps {
  /** Auth file name to inspect, or null when closed. */
  name: string | null;
  onClose: () => void;
}

/** Read-only drawer listing the models a given auth file can serve. */
export function AuthFileModelsDrawer({ name, onClose }: AuthFileModelsDrawerProps) {
  const { t } = useTranslation();
  const query = useAuthFileModelsQuery(name);
  const models = query.data ?? [];

  return (
    <Drawer open={name !== null} onOpenChange={(open) => !open && onClose()}>
      <DrawerContent>
        <DrawerHeader>
          <DrawerTitle>{t("auth_files.models_title")}</DrawerTitle>
          <DrawerDescription className="font-mono">{name}</DrawerDescription>
        </DrawerHeader>
        <DrawerBody>
          {query.isLoading ? (
            <div className="space-y-2">
              {Array.from({ length: 5 }).map((_, i) => (
                <Skeleton key={i} className="h-8 w-full" />
              ))}
            </div>
          ) : query.isError ? (
            <EmptyState
              tone="danger"
              title={t("table.error_title")}
              description={t("table.error_description")}
            />
          ) : models.length === 0 ? (
            <EmptyState
              title={t("auth_files.models_empty_title")}
              description={t("auth_files.models_empty_desc")}
            />
          ) : (
            <ul className="space-y-1">
              {models.map((model) => (
                <li
                  key={model.id}
                  className="flex items-center justify-between rounded-md border border-border bg-surface-sunken/40 px-3 py-2"
                >
                  <span className="font-mono text-xs text-foreground">{model.id}</span>
                  {model.owned_by ? (
                    <Badge variant="outline" className="text-[10px]">
                      {model.owned_by}
                    </Badge>
                  ) : null}
                </li>
              ))}
            </ul>
          )}
        </DrawerBody>
      </DrawerContent>
    </Drawer>
  );
}
