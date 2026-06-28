/**
 * Model aliases & exclusions API.
 *
 * Two related backend resources, both keyed by channel (a provider family such
 * as "claude", "gemini", "codex"):
 *
 *   oauth-model-alias: map<channel, Alias[]>
 *     GET    /oauth-model-alias              -> { "oauth-model-alias": map }
 *     PATCH  /oauth-model-alias  body: { channel, aliases: Alias[] }
 *              (empty aliases deletes the channel)
 *     DELETE /oauth-model-alias?channel=<c>
 *
 *   oauth-excluded-models: map<channel, string[]>
 *     GET    /oauth-excluded-models          -> { "oauth-excluded-models": map }
 *     PATCH  /oauth-excluded-models  body: { provider, models: string[] }
 *              (empty models deletes the provider)
 *     DELETE /oauth-excluded-models?provider=<p>
 */
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { managementApi } from "./client";
import { queryKeys } from "../query";

/** A single alias mapping a client-facing alias to an upstream model name. */
export interface ModelAlias {
  /** Upstream model name. */
  name: string;
  /** Client-facing alias that routes to `name`. */
  alias: string;
  /** Register the alias as an additional (forked) model rather than replacing. */
  fork?: boolean;
  /** Rewrite upstream response model fields back to the alias. */
  "force-mapping"?: boolean;
}

export type ModelAliasMap = Record<string, ModelAlias[]>;
export type ExcludedModelMap = Record<string, string[]>;

interface ModelAliasResponse {
  "oauth-model-alias"?: ModelAliasMap | null;
}
interface ExcludedModelsResponse {
  "oauth-excluded-models"?: ExcludedModelMap | null;
}

/** Fetch the channel -> aliases map. */
export function fetchModelAliases(signal?: AbortSignal): Promise<ModelAliasMap> {
  return managementApi
    .get<ModelAliasResponse>("/oauth-model-alias", signal ? { signal } : undefined)
    .then((res) => res["oauth-model-alias"] ?? {});
}

/** Replace (or delete, when empty) the aliases for a single channel. */
export function patchModelAliases(channel: string, aliases: ModelAlias[]): Promise<void> {
  return managementApi.patch<void>("/oauth-model-alias", { body: { channel, aliases } });
}

/** Delete an entire channel's aliases. */
export function deleteModelAliasChannel(channel: string): Promise<void> {
  return managementApi.delete<void>("/oauth-model-alias", { query: { channel } });
}

/** Fetch the channel -> excluded model ids map. */
export function fetchExcludedModels(signal?: AbortSignal): Promise<ExcludedModelMap> {
  return managementApi
    .get<ExcludedModelsResponse>("/oauth-excluded-models", signal ? { signal } : undefined)
    .then((res) => res["oauth-excluded-models"] ?? {});
}

/** Replace (or delete, when empty) the excluded models for a single provider. */
export function patchExcludedModels(provider: string, models: string[]): Promise<void> {
  return managementApi.patch<void>("/oauth-excluded-models", { body: { provider, models } });
}

/** Delete an entire provider's exclusion list. */
export function deleteExcludedProvider(provider: string): Promise<void> {
  return managementApi.delete<void>("/oauth-excluded-models", { query: { provider } });
}

/** Query hook for the model alias map. */
export function useModelAliasesQuery(enabled = true) {
  return useQuery({
    queryKey: queryKeys.modelAliases,
    queryFn: ({ signal }) => fetchModelAliases(signal),
    enabled,
  });
}

/** Mutation hook to set a channel's aliases. */
export function useUpdateModelAliasesMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ channel, aliases }: { channel: string; aliases: ModelAlias[] }) =>
      patchModelAliases(channel, aliases),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.modelAliases });
    },
  });
}

/** Mutation hook to delete a channel's aliases. */
export function useDeleteModelAliasChannelMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (channel: string) => deleteModelAliasChannel(channel),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.modelAliases });
    },
  });
}

/** Query hook for the excluded models map. */
export function useExcludedModelsQuery(enabled = true) {
  return useQuery({
    queryKey: queryKeys.excludedModels,
    queryFn: ({ signal }) => fetchExcludedModels(signal),
    enabled,
  });
}

/** Mutation hook to set a provider's excluded models. */
export function useUpdateExcludedModelsMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ provider, models }: { provider: string; models: string[] }) =>
      patchExcludedModels(provider, models),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.excludedModels });
    },
  });
}

/** Mutation hook to delete a provider's exclusion list. */
export function useDeleteExcludedProviderMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (provider: string) => deleteExcludedProvider(provider),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.excludedModels });
    },
  });
}
