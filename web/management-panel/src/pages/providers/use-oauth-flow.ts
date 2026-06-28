import * as React from "react";
import { useQueryClient } from "@tanstack/react-query";
import { ApiError } from "@/lib/api";
import {
  type OAuthProvider,
  fetchAuthStatus,
  requestAuthUrl,
} from "@/lib/api/oauth";
import { queryKeys } from "@/lib/query";

/** Lifecycle phase of an OAuth login attempt for a single provider. */
export type OAuthPhase = "idle" | "starting" | "waiting" | "success" | "error";

export interface OAuthFlowState {
  phase: OAuthPhase;
  /** Authorization URL once obtained (so the user can re-open the tab). */
  url: string | null;
  error: string | null;
}

const POLL_INTERVAL_MS = 2_000;
const POLL_TIMEOUT_MS = 5 * 60 * 1000;

const INITIAL: OAuthFlowState = { phase: "idle", url: null, error: null };

/**
 * Drives a provider OAuth login: requests the authorization URL, opens it in a
 * new tab, and polls `get-auth-status` until the backend reports success or
 * error. On success it invalidates the auth-files cache so the new credential
 * appears. Polling stops automatically on unmount or when the flow resolves.
 */
export function useOAuthFlow() {
  const queryClient = useQueryClient();
  const [states, setStates] = React.useState<Record<string, OAuthFlowState>>({});
  const timers = React.useRef<Record<string, ReturnType<typeof setTimeout>>>({});
  const mounted = React.useRef(true);

  React.useEffect(() => {
    mounted.current = true;
    const pending = timers.current;
    return () => {
      mounted.current = false;
      Object.values(pending).forEach((timer) => clearTimeout(timer));
    };
  }, []);

  const set = React.useCallback((provider: string, patch: Partial<OAuthFlowState>) => {
    setStates((prev) => ({ ...prev, [provider]: { ...(prev[provider] ?? INITIAL), ...patch } }));
  }, []);

  const poll = React.useCallback(
    (provider: OAuthProvider, state: string, deadline: number) => {
      const tick = async () => {
        if (!mounted.current) return;
        try {
          const result = await fetchAuthStatus(state);
          if (!mounted.current) return;
          if (result.status === "ok") {
            set(provider, { phase: "success", error: null });
            void queryClient.invalidateQueries({ queryKey: queryKeys.authFiles });
            return;
          }
          if (result.status === "error") {
            set(provider, { phase: "error", error: result.error ?? "Authentication failed" });
            return;
          }
        } catch (error) {
          if (!mounted.current) return;
          const message = error instanceof ApiError ? error.message : "Polling failed";
          set(provider, { phase: "error", error: message });
          return;
        }
        if (Date.now() > deadline) {
          set(provider, { phase: "error", error: "Timed out waiting for authorization" });
          return;
        }
        timers.current[provider] = setTimeout(() => void tick(), POLL_INTERVAL_MS);
      };
      timers.current[provider] = setTimeout(() => void tick(), POLL_INTERVAL_MS);
    },
    [queryClient, set],
  );

  const start = React.useCallback(
    async (provider: OAuthProvider) => {
      set(provider, { phase: "starting", error: null, url: null });
      try {
        const response = await requestAuthUrl(provider);
        if (!mounted.current) return;
        set(provider, { phase: "waiting", url: response.url });
        // Open in a new tab so the panel keeps polling in the background.
        window.open(response.url, "_blank", "noopener,noreferrer");
        poll(provider, response.state, Date.now() + POLL_TIMEOUT_MS);
      } catch (error) {
        if (!mounted.current) return;
        const message = error instanceof ApiError ? error.message : "Failed to start login";
        set(provider, { phase: "error", error: message });
      }
    },
    [poll, set],
  );

  const reset = React.useCallback(
    (provider: OAuthProvider) => {
      const timer = timers.current[provider];
      if (timer) clearTimeout(timer);
      set(provider, { ...INITIAL });
    },
    [set],
  );

  const getState = React.useCallback(
    (provider: OAuthProvider): OAuthFlowState => states[provider] ?? INITIAL,
    [states],
  );

  return { start, reset, getState };
}
