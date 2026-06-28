/**
 * Config API — visual config editor, raw YAML, and downstream API keys.
 *
 * Backend contract:
 *   GET  /config       -> ManagementConfig (JSON)
 *   PUT  /config       body: Partial<ManagementConfig>
 *   GET  /config.yaml  -> raw YAML text (text/plain)
 *   PUT  /config.yaml  body: raw YAML string (Content-Type: text/plain)
 *   GET  /api-keys     -> { "api-keys": string[] }
 *   PUT  /api-keys     body: { "api-keys": string[] }
 */
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { managementApi, MANAGEMENT_PREFIX, apiClient } from "./client";
import type { ManagementConfig, ApiKeysResponse } from "./types";
import { queryKeys } from "@/lib/query";

// ---------------------------------------------------------------------------
// Fetch helpers
// ---------------------------------------------------------------------------

/** Read structured config (JSON). */
export function fetchConfig(signal?: AbortSignal): Promise<ManagementConfig> {
  return managementApi.get<ManagementConfig>("/config", signal ? { signal } : undefined);
}

/** Write partial config (JSON merge). */
export function putConfig(config: Partial<ManagementConfig>): Promise<void> {
  return managementApi.put<void>("/config", { body: config });
}

/** Read the raw YAML config file. */
export function fetchConfigYaml(signal?: AbortSignal): Promise<string> {
  return apiClient.get<string>(`${MANAGEMENT_PREFIX}/config.yaml`, {
    ...(signal ? { signal } : {}),
    headers: { Accept: "text/plain" },
  });
}

/** Write the raw YAML config file. */
export function putConfigYaml(yaml: string): Promise<void> {
  return apiClient.put<void>(`${MANAGEMENT_PREFIX}/config.yaml`, {
    body: yaml,
    headers: { "Content-Type": "text/plain" },
  });
}

/** Read downstream API keys. */
export function fetchDownstreamApiKeys(signal?: AbortSignal): Promise<ApiKeysResponse> {
  return managementApi.get<ApiKeysResponse>("/api-keys", signal ? { signal } : undefined);
}

/** Write the full downstream API key list. */
export function putDownstreamApiKeys(keys: string[]): Promise<void> {
  return managementApi.put<void>("/api-keys", { body: { "api-keys": keys } });
}

// ---------------------------------------------------------------------------
// React Query hooks
// ---------------------------------------------------------------------------

export function useManagementConfigQuery(enabled = true) {
  return useQuery({
    queryKey: queryKeys.config,
    queryFn: ({ signal }) => fetchConfig(signal),
    enabled,
  });
}

export function usePutConfigMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (config: Partial<ManagementConfig>) => putConfig(config),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.config });
      void queryClient.invalidateQueries({ queryKey: queryKeys.configYaml });
    },
  });
}

export function useConfigYamlQuery(enabled = true) {
  return useQuery({
    queryKey: queryKeys.configYaml,
    queryFn: ({ signal }) => fetchConfigYaml(signal),
    enabled,
  });
}

export function usePutConfigYamlMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (yaml: string) => putConfigYaml(yaml),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.configYaml });
      void queryClient.invalidateQueries({ queryKey: queryKeys.config });
    },
  });
}

export function useDownstreamApiKeysQuery(enabled = true) {
  return useQuery({
    queryKey: queryKeys.downstreamApiKeys,
    queryFn: ({ signal }) => fetchDownstreamApiKeys(signal),
    enabled,
  });
}

export function usePutDownstreamApiKeysMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (keys: string[]) => putDownstreamApiKeys(keys),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.downstreamApiKeys });
    },
  });
}
