import { useQuery } from "@tanstack/react-query";
import { queryKeys } from "./query";
import {
  fetchConfig,
  fetchHealth,
  fetchLatestVersion,
  fetchUsageStatistics,
  fetchApiKeys,
} from "./api";

/** Backend liveness, polled so the header pill stays current. */
export function useHealthQuery() {
  return useQuery({
    queryKey: queryKeys.health,
    queryFn: ({ signal }) => fetchHealth(signal),
    refetchInterval: 30_000,
    staleTime: 15_000,
    retry: 1,
  });
}

/** Latest upstream release tag (best-effort; failures are non-fatal). */
export function useLatestVersionQuery(enabled = true) {
  return useQuery({
    queryKey: queryKeys.latestVersion,
    queryFn: ({ signal }) => fetchLatestVersion(signal),
    enabled,
    staleTime: 60 * 60 * 1000,
    retry: 0,
  });
}

/** Full management configuration. */
export function useConfigQuery(enabled = true) {
  return useQuery({
    queryKey: queryKeys.config,
    queryFn: ({ signal }) => fetchConfig(signal),
    enabled,
  });
}

/** Aggregate usage statistics (PPAP private feature). */
export function useUsageStatisticsQuery(enabled = true) {
  return useQuery({
    queryKey: queryKeys.usageStatistics,
    queryFn: ({ signal }) => fetchUsageStatistics(signal),
    enabled,
    refetchInterval: 60_000,
  });
}

/** Proxy-service API keys. */
export function useApiKeysQuery(enabled = true) {
  return useQuery({
    queryKey: queryKeys.apiKeys,
    queryFn: ({ signal }) => fetchApiKeys(signal),
    enabled,
  });
}
