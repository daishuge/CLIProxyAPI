/**
 * OAuth login flows API.
 *
 * Each provider exposes a GET endpoint that returns an authorization URL plus a
 * state token. The panel opens the URL in a new tab; the backend runs a local
 * callback forwarder (enabled by `is_webui=1`) and completes the exchange. The
 * panel polls `get-auth-status?state=<state>` until it resolves.
 *
 * Backend contract (internal/api/handlers/management/auth_files.go):
 *   GET /{provider}-auth-url?is_webui=1 -> { status, url, state }
 *   GET /get-auth-status?state=<state>  -> { status: "ok"|"wait"|"error", error? }
 */
import { managementApi } from "./client";

/** OAuth providers that expose a management auth-url endpoint. */
export const OAUTH_PROVIDERS = [
  "anthropic",
  "codex",
  "gemini",
  "kimi",
  "xai",
  "antigravity",
] as const;
export type OAuthProvider = (typeof OAUTH_PROVIDERS)[number];

/** Providers that the backend currently exposes a dedicated auth-url route for. */
export const OAUTH_PROVIDER_ENDPOINTS: Record<OAuthProvider, string> = {
  anthropic: "/anthropic-auth-url",
  codex: "/codex-auth-url",
  gemini: "/gemini-auth-url",
  kimi: "/kimi-auth-url",
  xai: "/xai-auth-url",
  antigravity: "/antigravity-auth-url",
};

export interface AuthUrlResponse {
  status?: string;
  url: string;
  state: string;
}

export type AuthStatus = "ok" | "wait" | "error";

export interface AuthStatusResponse {
  status: AuthStatus;
  error?: string;
}

/** Request the authorization URL + state for a provider's OAuth flow. */
export function requestAuthUrl(
  provider: OAuthProvider,
  signal?: AbortSignal,
): Promise<AuthUrlResponse> {
  return managementApi.get<AuthUrlResponse>(OAUTH_PROVIDER_ENDPOINTS[provider], {
    query: { is_webui: "1" },
    ...(signal ? { signal } : {}),
  });
}

/** Poll the status of an in-flight OAuth session. */
export function fetchAuthStatus(
  state: string,
  signal?: AbortSignal,
): Promise<AuthStatusResponse> {
  return managementApi.get<AuthStatusResponse>("/get-auth-status", {
    query: { state },
    ...(signal ? { signal } : {}),
  });
}
