/**
 * System API — health, version, and auth status.
 *
 * Backend contract:
 *   GET /healthz                       -> { status: string }
 *   GET /v0/management/latest-version  -> { "latest-version": string }
 *   GET /v0/management/get-auth-status -> auth status object
 */
import { useQuery } from "@tanstack/react-query";
import { managementApi } from "./client";
import { queryKeys } from "@/lib/query";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface AuthStatus {
  /** Number of active provider connections. */
  "active-providers"?: number;
  /** Total auth file count. */
  "total-auth-files"?: number;
  /** Additional status fields the backend may include. */
  [key: string]: unknown;
}

// ---------------------------------------------------------------------------
// Fetch helpers
// ---------------------------------------------------------------------------

export function fetchAuthStatus(signal?: AbortSignal): Promise<AuthStatus> {
  return managementApi.get<AuthStatus>("/get-auth-status", signal ? { signal } : undefined);
}

// ---------------------------------------------------------------------------
// React Query hooks
// ---------------------------------------------------------------------------

/** Auth status — provider & auth file counts. */
export function useAuthStatusQuery(enabled = true) {
  return useQuery({
    queryKey: queryKeys.authStatus,
    queryFn: ({ signal }) => fetchAuthStatus(signal),
    enabled,
  });
}
