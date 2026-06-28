/**
 * Plugins API — installed plugin management and plugin store.
 *
 * Backend contract:
 *   GET  /plugins        -> { plugins: Plugin[] }
 *   PUT  /plugins        body: { name, enabled }
 *   GET  /plugin-store   -> { plugins: StorePlugin[] }
 *   POST /plugin-store/install  body: { name }
 */
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { managementApi } from "./client";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface Plugin {
  name: string;
  version: string;
  enabled: boolean;
  description: string;
  config?: Record<string, unknown>;
}

export interface PluginsResponse {
  plugins: Plugin[];
}

export interface StorePlugin {
  name: string;
  description: string;
  version: string;
  installed: boolean;
}

export interface PluginStoreResponse {
  plugins: StorePlugin[];
}

// ---------------------------------------------------------------------------
// Query keys (local string literals — no centralized registry dependency)
// ---------------------------------------------------------------------------

const PLUGINS_KEY = ["plugins"] as const;
const PLUGIN_STORE_KEY = ["plugin-store"] as const;

// ---------------------------------------------------------------------------
// Fetch helpers
// ---------------------------------------------------------------------------

export function fetchPlugins(signal?: AbortSignal): Promise<PluginsResponse> {
  return managementApi.get<PluginsResponse>("/plugins", signal ? { signal } : undefined);
}

export function updatePlugin(name: string, patch: { enabled: boolean }): Promise<void> {
  return managementApi.put<void>("/plugins", { body: { name, enabled: patch.enabled } });
}

export function fetchPluginStore(signal?: AbortSignal): Promise<PluginStoreResponse> {
  return managementApi.get<PluginStoreResponse>("/plugin-store", signal ? { signal } : undefined);
}

export function installPlugin(name: string): Promise<void> {
  return managementApi.post<void>("/plugin-store/install", { body: { name } });
}

// ---------------------------------------------------------------------------
// React Query hooks
// ---------------------------------------------------------------------------

export function usePluginsQuery(enabled = true) {
  return useQuery({
    queryKey: PLUGINS_KEY,
    queryFn: ({ signal }) => fetchPlugins(signal),
    enabled,
  });
}

export function useUpdatePluginMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ name, enabled }: { name: string; enabled: boolean }) =>
      updatePlugin(name, { enabled }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: PLUGINS_KEY });
    },
  });
}

export function usePluginStoreQuery(enabled = true) {
  return useQuery({
    queryKey: PLUGIN_STORE_KEY,
    queryFn: ({ signal }) => fetchPluginStore(signal),
    enabled,
  });
}

export function useInstallPluginMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (name: string) => installPlugin(name),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: PLUGIN_STORE_KEY });
      void queryClient.invalidateQueries({ queryKey: PLUGINS_KEY });
    },
  });
}
