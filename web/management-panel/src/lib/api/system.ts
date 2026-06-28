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
import { fetchHealth, fetchLatestVersion } from "./endpoints";

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
// Query keys (local string literals — no centralized registry dependency)
// ---------------------------------------------------------------------------

const HEALTH_KEY = ["health"] as const;
const LATEST_VERSION_KEY = ["latest-version"] as const;
const AUTH_STATUS_KEY = ["auth-status"] as const;

// ---------------------------------------------------------------------------
// Fetch helpers
// ---------------------------------------------------------------------------

export function fetchAuthStatus(signal?: AbortSignal): Promise<AuthStatus> {
  return managementApi.get<AuthStatus>("/get-auth-status", signal ? { signal } : undefined);
}

// ---------------------------------------------------------------------------
// React Query hooks
// ---------------------------------------------------------------------------

/** Health with automatic polling (30 s). Wraps the shared fetchHealth. */
export function useSystemHealthQuery() {
  return useQuery({
    queryKey: HEALTH_KEY,
    queryFn: ({ signal }) => fetchHealth(signal),
    refetchInterval: 30_000,
    staleTime: 15_000,
    retry: 1,
  });
}

/** Latest upstream version. Wraps the shared fetchLatestVersion. */
export function useSystemLatestVersionQuery(enabled = true) {
  return useQuery({
    queryKey: LATEST_VERSION_KEY,
    queryFn: ({ signal }) => fetchLatestVersion(signal),
    enabled,
    staleTime: 60 * 60 * 1000,
    retry: 0,
  });
}

/** Auth status — provider & auth file counts. */
export function useAuthStatusQuery(enabled = true) {
  return useQuery({
    queryKey: AUTH_STATUS_KEY,
    queryFn: ({ signal }) => fetchAuthStatus(signal),
    enabled,
  });
}
