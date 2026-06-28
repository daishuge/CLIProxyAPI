/**
 * Traffic & Usage API — hooks and fetch helpers for usage statistics,
 * export/import, and per-API-key consumption data.
 */
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { managementApi } from "./client";
import { queryKeys } from "../query";
import type { UsageStatisticsResponse } from "./types";
import { fetchUsageStatistics } from "./endpoints";

// ===== Types =====

/** Per-API-key usage rollup from GET /api-key-usage. */
export interface ApiKeyUsageEntry {
  requests: number;
  total_tokens: number;
  total_input_tokens: number;
  total_cached_tokens: number;
  cache_hit_rate: number;
  average_latency_ms: number;
  tps: number;
}

interface ApiKeyUsageResponse {
  "api-key-usage"?: Record<string, ApiKeyUsageEntry>;
}

// ===== Fetch functions =====

/** Re-export the existing statistics query for co-location. */
export { fetchUsageStatistics };

/** Export usage statistics as a JSON blob (raw response for downloadBlob). */
export async function exportUsageStatistics(): Promise<Response> {
  return managementApi.get<Response>("/usage-statistics/export", { raw: true });
}

/** Import usage statistics from a JSON file. */
export async function importUsageStatistics(file: File): Promise<void> {
  const formData = new FormData();
  formData.append("file", file);
  return managementApi.post<void>("/usage-statistics/import", { body: formData });
}

/** Fetch per-API-key usage breakdown. */
export function fetchApiKeyUsage(signal?: AbortSignal): Promise<Record<string, ApiKeyUsageEntry>> {
  return managementApi
    .get<ApiKeyUsageResponse>("/api-key-usage", signal ? { signal } : undefined)
    .then((res) => res["api-key-usage"] ?? {});
}

// ===== React Query hooks =====

/** Aggregate usage statistics (wraps existing endpoint). */
export function useUsageStatisticsQuery(enabled = true) {
  return useQuery<UsageStatisticsResponse>({
    queryKey: queryKeys.usageStatistics,
    queryFn: ({ signal }) => fetchUsageStatistics(signal),
    enabled,
    refetchInterval: 60_000,
  });
}

/** Export mutation — triggers download on success. */
export function useExportUsageStatisticsMutation() {
  return useMutation({
    mutationFn: () => exportUsageStatistics(),
  });
}

/** Import mutation — invalidates usage queries on success. */
export function useImportUsageStatisticsMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (file: File) => importUsageStatistics(file),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.usageStatistics });
      void queryClient.invalidateQueries({ queryKey: queryKeys.apiKeyUsage });
    },
  });
}

/** Per-API-key usage breakdown. */
export function useApiKeyUsageQuery(enabled = true) {
  return useQuery({
    queryKey: queryKeys.apiKeyUsage,
    queryFn: ({ signal }) => fetchApiKeyUsage(signal),
    enabled,
  });
}
