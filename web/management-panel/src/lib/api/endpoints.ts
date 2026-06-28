/**
 * Endpoint helpers — thin, typed wrappers around the management API client.
 * Feature pages consume these through React Query hooks (see `query.ts`).
 */
import { apiClient, managementApi, HEALTH_PATH } from "./client";
import type {
  ApiKeysResponse,
  HealthResponse,
  LatestVersionResponse,
  ManagementConfig,
  UsageStatisticsResponse,
} from "./types";

/** Unauthenticated liveness probe. */
export function fetchHealth(signal?: AbortSignal): Promise<HealthResponse> {
  return apiClient.get<HealthResponse>(HEALTH_PATH, {
    timeoutMs: 8_000,
    ...(signal ? { signal } : {}),
  });
}

/** Full management configuration; also the auth-probe target on login. */
export function fetchConfig(signal?: AbortSignal): Promise<ManagementConfig> {
  return managementApi.get<ManagementConfig>("/config", signal ? { signal } : undefined);
}

/** Aggregate usage statistics (PPAP private feature). */
export function fetchUsageStatistics(signal?: AbortSignal): Promise<UsageStatisticsResponse> {
  return managementApi.get<UsageStatisticsResponse>(
    "/usage-statistics",
    signal ? { signal } : undefined,
  );
}

/** Latest upstream release tag (proxied through the backend). */
export function fetchLatestVersion(signal?: AbortSignal): Promise<LatestVersionResponse> {
  return managementApi.get<LatestVersionResponse>(
    "/latest-version",
    signal ? { signal } : undefined,
  );
}

/** Proxy-service API keys. */
export function fetchApiKeys(signal?: AbortSignal): Promise<ApiKeysResponse> {
  return managementApi.get<ApiKeysResponse>("/api-keys", signal ? { signal } : undefined);
}
