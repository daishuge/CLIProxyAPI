import * as React from "react";
import { Route, RefreshCw } from "lucide-react";
import { useTranslation } from "react-i18next";
import { ApiError } from "@/lib/api";
import {
  ROUTING_STRATEGIES,
  type RoutingStrategy,
  useRoutingStrategyQuery,
  useUpdateRoutingStrategyMutation,
} from "@/lib/api/routing";
import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  EmptyState,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Skeleton,
  toast,
} from "@/components/ui";

function isKnownStrategy(value: string): value is RoutingStrategy {
  return (ROUTING_STRATEGIES as readonly string[]).includes(value);
}

/** Read/write the global routing strategy (round-robin vs fill-first). */
export function RoutingStrategyTab() {
  const { t } = useTranslation();
  const query = useRoutingStrategyQuery();
  const mutation = useUpdateRoutingStrategyMutation();

  const [selected, setSelected] = React.useState<RoutingStrategy>("round-robin");

  React.useEffect(() => {
    if (query.data && isKnownStrategy(query.data)) {
      setSelected(query.data);
    }
  }, [query.data]);

  const dirty = query.data !== selected;

  const save = () => {
    mutation.mutate(selected, {
      onSuccess: () => toast.success(t("routing_strategy.saved")),
      onError: (error) => {
        const message = error instanceof ApiError ? error.message : t("common.unknown_error");
        toast.error(t("routing_strategy.save_failed"), message);
      },
    });
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Route className="size-5 text-muted-foreground" />
          {t("routing_strategy.title")}
        </CardTitle>
        <CardDescription>{t("routing_strategy.subtitle")}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {query.isLoading ? (
          <Skeleton className="h-9 w-full max-w-sm" />
        ) : query.isError ? (
          <EmptyState
            tone="danger"
            title={t("table.error_title")}
            description={t("table.error_description")}
            action={
              <Button variant="outline" size="sm" onClick={() => void query.refetch()}>
                <RefreshCw className="size-4" />
                {t("common.retry")}
              </Button>
            }
          />
        ) : (
          <>
            <div className="grid max-w-sm gap-2">
              <Select value={selected} onValueChange={(v) => setSelected(v as RoutingStrategy)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {ROUTING_STRATEGIES.map((strategy) => (
                    <SelectItem key={strategy} value={strategy}>
                      {t(`routing_strategy.option_${strategy.replace("-", "_")}`)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <p className="text-xs text-muted-foreground">
                {t(`routing_strategy.desc_${selected.replace("-", "_")}`)}
              </p>
            </div>
            <Button onClick={save} loading={mutation.isPending} disabled={!dirty}>
              {t("common.save")}
            </Button>
          </>
        )}
      </CardContent>
    </Card>
  );
}
