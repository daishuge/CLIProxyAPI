/**
 * Custom Upstreams API — PPAP's headline feature. Custom upstreams are the
 * management-facing name for the backend `openai-compatibility` provider pool:
 * the same `config.OpenAICompatibility` records are served under both
 * `/custom-upstreams` and `/openai-compatibility`, so this module is the single
 * entry point for editing OpenAI-compatible upstream definitions.
 *
 * Backend contract (internal/api/handlers/management/config_lists.go):
 *   GET    /custom-upstreams                  -> { "custom-upstreams": Upstream[] }
 *   PUT    /custom-upstreams   body: Upstream[]  (full replace; entries with an
 *                                                  empty base-url are dropped)
 *   PATCH  /custom-upstreams   body: { name|index, value: Partial<Upstream> }
 *                                                  (partial update; clearing
 *                                                  base-url deletes the entry)
 *   DELETE /custom-upstreams?name=<name>      (or ?index=<n>)
 */
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { managementApi } from "./client";
import { queryKeys } from "../query";

/** Discrete thinking/reasoning capability descriptor for a model. Preserved
 * opaquely across edits — the panel does not expose its internals yet. */
export interface ThinkingSupport {
  min?: number;
  max?: number;
  zero_allowed?: boolean;
  dynamic_allowed?: boolean;
  levels?: string[];
}

/** A single API key entry with an optional per-key proxy override. */
export interface UpstreamApiKeyEntry {
  "api-key": string;
  "proxy-url"?: string;
}

/** A model mapping exposed by a custom upstream. */
export interface UpstreamModel {
  name: string;
  alias: string;
  "force-mapping"?: boolean;
  image?: boolean;
  thinking?: ThinkingSupport | null;
}

/** A custom upstream (OpenAI-compatible provider) definition. */
export interface CustomUpstream {
  name: string;
  priority?: number;
  disabled?: boolean;
  prefix?: string;
  "base-url": string;
  "api-key-entries"?: UpstreamApiKeyEntry[];
  models: UpstreamModel[];
  headers?: Record<string, string>;
  "disable-cooling"?: boolean;
  /** Read-only, attached by the backend; never sent on write. */
  auth_index?: string;
}

interface CustomUpstreamsResponse {
  "custom-upstreams"?: CustomUpstream[];
}

/** List all custom upstreams. */
export function fetchCustomUpstreams(signal?: AbortSignal): Promise<CustomUpstream[]> {
  return managementApi
    .get<CustomUpstreamsResponse>("/custom-upstreams", signal ? { signal } : undefined)
    .then((res) => res["custom-upstreams"] ?? []);
}

/** Replace the entire custom-upstreams list. */
export function putCustomUpstreams(upstreams: CustomUpstream[]): Promise<void> {
  return managementApi.put<void>("/custom-upstreams", { body: upstreams });
}

/** Patch a single upstream identified by its current name. */
export function patchCustomUpstream(
  name: string,
  value: Partial<CustomUpstream>,
): Promise<void> {
  return managementApi.patch<void>("/custom-upstreams", { body: { name, value } });
}

/** Delete a single upstream by name. */
export function deleteCustomUpstream(name: string): Promise<void> {
  return managementApi.delete<void>("/custom-upstreams", { query: { name } });
}

/** Query hook for the custom-upstreams list. */
export function useCustomUpstreamsQuery(enabled = true) {
  return useQuery({
    queryKey: queryKeys.customUpstreams,
    queryFn: ({ signal }) => fetchCustomUpstreams(signal),
    enabled,
  });
}

/**
 * Create a new upstream. There is no dedicated create endpoint, so the panel
 * appends to the current list and PUTs the whole collection. Callers pass the
 * existing list to avoid a read-modify-write race with stale cache data.
 */
export function useCreateCustomUpstreamMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ existing, upstream }: { existing: CustomUpstream[]; upstream: CustomUpstream }) =>
      putCustomUpstreams([...existing, upstream]),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.customUpstreams });
    },
  });
}

/** Update an existing upstream by name via PATCH. */
export function useUpdateCustomUpstreamMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ name, value }: { name: string; value: Partial<CustomUpstream> }) =>
      patchCustomUpstream(name, value),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.customUpstreams });
    },
  });
}

/** Delete an upstream by name. */
export function useDeleteCustomUpstreamMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (name: string) => deleteCustomUpstream(name),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.customUpstreams });
    },
  });
}
