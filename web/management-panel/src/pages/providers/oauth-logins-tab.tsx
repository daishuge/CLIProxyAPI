import { CheckCircle2, ExternalLink, LogIn, Loader2, XCircle } from "lucide-react";
import { useTranslation } from "react-i18next";
import { OAUTH_PROVIDERS, type OAuthProvider } from "@/lib/api/oauth";
import {
  Badge,
  Button,
  Card,
  CardContent,
} from "@/components/ui";
import { useOAuthFlow } from "./use-oauth-flow";

/**
 * OAuth Logins. One card per provider; starting a flow requests an
 * authorization URL, opens it in a new tab and polls until the backend
 * completes the credential exchange. Live phase is reflected per card.
 */
export function OAuthLoginsTab() {
  const { t } = useTranslation();
  const { start, reset, getState } = useOAuthFlow();

  return (
    <div className="space-y-4">
      <p className="text-sm text-muted-foreground">{t("oauth.subtitle")}</p>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
        {OAUTH_PROVIDERS.map((provider: OAuthProvider) => {
          const state = getState(provider);
          const busy = state.phase === "starting" || state.phase === "waiting";
          return (
            <Card key={provider}>
              <CardContent className="flex flex-col gap-3 p-4">
                <div className="flex items-center justify-between">
                  <span className="font-medium capitalize text-foreground">
                    {t(`oauth.provider_${provider}`, { defaultValue: provider })}
                  </span>
                  {state.phase === "success" ? (
                    <Badge variant="success" dot>
                      {t("oauth.status_connected")}
                    </Badge>
                  ) : state.phase === "error" ? (
                    <Badge variant="danger" dot>
                      {t("oauth.status_failed")}
                    </Badge>
                  ) : state.phase === "waiting" ? (
                    <Badge variant="warning" dot>
                      {t("oauth.status_waiting")}
                    </Badge>
                  ) : null}
                </div>

                {state.phase === "error" && state.error ? (
                  <p className="flex items-start gap-1.5 text-xs text-danger">
                    <XCircle className="mt-0.5 size-3.5 shrink-0" />
                    <span className="break-words">{state.error}</span>
                  </p>
                ) : state.phase === "success" ? (
                  <p className="flex items-center gap-1.5 text-xs text-success">
                    <CheckCircle2 className="size-3.5" />
                    {t("oauth.success_desc")}
                  </p>
                ) : state.phase === "waiting" ? (
                  <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
                    <Loader2 className="size-3.5 animate-[ppap-spin_0.7s_linear_infinite]" />
                    {t("oauth.waiting_desc")}
                  </p>
                ) : (
                  <p className="text-xs text-muted-foreground">{t("oauth.idle_desc")}</p>
                )}

                <div className="mt-1 flex items-center gap-2">
                  {state.phase === "success" || state.phase === "error" ? (
                    <Button variant="outline" size="sm" onClick={() => reset(provider)}>
                      {t("oauth.retry")}
                    </Button>
                  ) : (
                    <Button size="sm" onClick={() => void start(provider)} loading={busy}>
                      <LogIn className="size-4" />
                      {t("oauth.connect")}
                    </Button>
                  )}
                  {state.url && state.phase === "waiting" ? (
                    <Button variant="ghost" size="sm" asChild>
                      <a href={state.url} target="_blank" rel="noopener noreferrer">
                        <ExternalLink className="size-4" />
                        {t("oauth.reopen")}
                      </a>
                    </Button>
                  ) : null}
                </div>
              </CardContent>
            </Card>
          );
        })}
      </div>
    </div>
  );
}
