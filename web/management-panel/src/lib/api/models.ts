/**
 * Models catalog API.
 *
 * The backend exposes static model definitions per channel (provider family):
 *   GET /model-definitions/:channel -> { channel, models: ModelInfo[] }
 *
 * There is no single flat "/models" endpoint; the catalog is browsed by
 * selecting a channel. Known channels are enumerated in
 * internal/registry/model_definitions.go.
 *
 * Force-model-prefix is a global toggle that namespaces every model id by its
 * provider prefix:
 *   GET /force-model-prefix -> { "force-model-prefix": boolean }
 *   PUT /force-model-prefix   body: { "value": boolean }
 */
import { useMutation, useQueries, useQuery, useQueryClient } from "@tanstack/react-query";
import { managementApi } from "./client";
import { queryKeys } from "../query";

/** Channels (provider families) that expose static model definitions. */
export const MODEL_CHANNELS = [
  "claude",
  "codex",
  "gemini",
  "vertex",
  "aistudio",
  "kimi",
  "antigravity",
  "xai",
] as const;
export type ModelChannel = (typeof MODEL_CHANNELS)[number];

/** A model entry as returned by the registry. Extra fields are tolerated. */
export interface ModelInfo {
  id: string;
  object?: string;
  display_name?: string;
  type?: string;
  owned_by?: string;
  [key: string]: unknown;
}

interface ModelDefinitionsResponse {
  channel?: string;
  models?: ModelInfo[];
}

/** A model row flattened with its owning channel, for catalog browsing. */
export interface CatalogModel extends ModelInfo {
  channel: ModelChannel;
}

/** Fetch static model definitions for a single channel. */
export function fetchModelDefinitions(
  channel: ModelChannel,
  signal?: AbortSignal,
): Promise<ModelInfo[]> {
  return managementApi
    .get<ModelDefinitionsResponse>(
      `/model-definitions/${encodeURIComponent(channel)}`,
      signal ? { signal } : undefined,
    )
    .then((res) => res.models ?? []);
}

/** Query a single channel's model definitions. */
export function useModelDefinitionsQuery(channel: ModelChannel, enabled = true) {
  return useQuery({
    queryKey: queryKeys.modelDefinitions(channel),
    queryFn: ({ signal }) => fetchModelDefinitions(channel, signal),
    enabled,
    staleTime: 5 * 60 * 1000,
  });
}

/**
 * Browse the full catalog by fanning out across every known channel. Failed
 * channels are skipped so one provider error does not blank the whole table.
 */
export function useModelCatalogQueries() {
  const queries = useQueries({
    queries: MODEL_CHANNELS.map((channel) => ({
      queryKey: queryKeys.modelDefinitions(channel),
      queryFn: ({ signal }: { signal: AbortSignal }) => fetchModelDefinitions(channel, signal),
      staleTime: 5 * 60 * 1000,
    })),
  });

  const models: CatalogModel[] = [];
  queries.forEach((query, index) => {
    const channel = MODEL_CHANNELS[index]!;
    for (const model of query.data ?? []) {
      models.push({ ...model, channel });
    }
  });

  return {
    models,
    isLoading: queries.some((q) => q.isLoading),
    isError: queries.length > 0 && queries.every((q) => q.isError),
    refetch: () => {
      queries.forEach((q) => void q.refetch());
    },
  };
}

interface ForceModelPrefixResponse {
  "force-model-prefix"?: boolean;
}

/** Read the global force-model-prefix toggle. */
export function fetchForceModelPrefix(signal?: AbortSignal): Promise<boolean> {
  return managementApi
    .get<ForceModelPrefixResponse>("/force-model-prefix", signal ? { signal } : undefined)
    .then((res) => res["force-model-prefix"] ?? false);
}

/** Write the global force-model-prefix toggle. */
export function putForceModelPrefix(value: boolean): Promise<void> {
  return managementApi.put<void>("/force-model-prefix", { body: { value } });
}

/** Query hook for force-model-prefix. */
export function useForceModelPrefixQuery(enabled = true) {
  return useQuery({
    queryKey: queryKeys.forceModelPrefix,
    queryFn: ({ signal }) => fetchForceModelPrefix(signal),
    enabled,
  });
}

/** Mutation hook for force-model-prefix. */
export function useUpdateForceModelPrefixMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (value: boolean) => putForceModelPrefix(value),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.forceModelPrefix });
    },
  });
}
