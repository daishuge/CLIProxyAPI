/**
 * Provider API Keys API — direct API-key credentials for the four key-based
 * provider families. All four share a common CRUD protocol:
 *
 *   GET    /{provider}-api-key             -> { "{provider}-api-key": Key[] }
 *   PUT    /{provider}-api-key   body: Key[]                       (full replace)
 *   PATCH  /{provider}-api-key   body: { index|match, value: Partial<Key> }
 *   DELETE /{provider}-api-key?api-key=<k>[&base-url=<u>]
 *
 * `match` / the `api-key` query param identify an entry by its key value;
 * `base-url` disambiguates when several entries share an api-key. Sending an
 * empty `api-key` in a PATCH value deletes the entry (gemini/vertex).
 */
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { managementApi } from "./client";
import { queryKeys } from "../query";

/** Key-based provider families. */
export const PROVIDER_KEY_TYPES = ["claude", "codex", "gemini", "vertex"] as const;
export type ProviderKeyType = (typeof PROVIDER_KEY_TYPES)[number];

/** A model mapping carried by claude/codex keys. */
export interface ProviderKeyModel {
  name: string;
  alias: string;
  "force-mapping"?: boolean;
}

/** A provider API key entry. Optional fields apply per provider family. */
export interface ProviderKey {
  "api-key": string;
  priority?: number;
  prefix?: string;
  "base-url"?: string;
  "proxy-url"?: string;
  headers?: Record<string, string>;
  "excluded-models"?: string[];
  /** claude & codex only. */
  models?: ProviderKeyModel[];
  /** claude only. */
  "rebuild-mid-system-message"?: boolean;
  /** Read-only, attached by the backend. */
  auth_index?: string;
}

const RESOURCE: Record<ProviderKeyType, string> = {
  claude: "/claude-api-key",
  codex: "/codex-api-key",
  gemini: "/gemini-api-key",
  vertex: "/vertex-api-key",
};

const RESPONSE_KEY: Record<ProviderKeyType, string> = {
  claude: "claude-api-key",
  codex: "codex-api-key",
  gemini: "gemini-api-key",
  vertex: "vertex-api-key",
};

/** Provider families that support per-key model mappings. */
export const PROVIDERS_WITH_MODELS: ReadonlySet<ProviderKeyType> = new Set(["claude", "codex"]);

/** List a provider's API keys. */
export function fetchProviderKeys(
  provider: ProviderKeyType,
  signal?: AbortSignal,
): Promise<ProviderKey[]> {
  return managementApi
    .get<Record<string, ProviderKey[] | undefined>>(
      RESOURCE[provider],
      signal ? { signal } : undefined,
    )
    .then((res) => res[RESPONSE_KEY[provider]] ?? []);
}

/** Replace the entire key list for a provider. */
export function putProviderKeys(provider: ProviderKeyType, keys: ProviderKey[]): Promise<void> {
  return managementApi.put<void>(RESOURCE[provider], { body: keys });
}

/** Patch a single key identified by its current api-key value. */
export function patchProviderKey(
  provider: ProviderKeyType,
  match: string,
  value: Partial<ProviderKey>,
): Promise<void> {
  return managementApi.patch<void>(RESOURCE[provider], { body: { match, value } });
}

/** Delete a key by its api-key value, optionally disambiguated by base-url. */
export function deleteProviderKey(
  provider: ProviderKeyType,
  apiKey: string,
  baseUrl?: string,
): Promise<void> {
  return managementApi.delete<void>(RESOURCE[provider], {
    query: { "api-key": apiKey, ...(baseUrl !== undefined ? { "base-url": baseUrl } : {}) },
  });
}

/** Query hook for a provider's keys. */
export function useProviderKeysQuery(provider: ProviderKeyType, enabled = true) {
  return useQuery({
    queryKey: queryKeys.providerKeys(provider),
    queryFn: ({ signal }) => fetchProviderKeys(provider, signal),
    enabled,
  });
}

/** Append a new key by PUTting the existing list plus the new entry. */
export function useCreateProviderKeyMutation(provider: ProviderKeyType) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ existing, key }: { existing: ProviderKey[]; key: ProviderKey }) =>
      putProviderKeys(provider, [...existing, key]),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.providerKeys(provider) });
    },
  });
}

/** Update an existing key via PATCH (matched by its previous api-key value). */
export function useUpdateProviderKeyMutation(provider: ProviderKeyType) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ match, value }: { match: string; value: Partial<ProviderKey> }) =>
      patchProviderKey(provider, match, value),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.providerKeys(provider) });
    },
  });
}

/** Delete a key. */
export function useDeleteProviderKeyMutation(provider: ProviderKeyType) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ apiKey, baseUrl }: { apiKey: string; baseUrl?: string }) =>
      deleteProviderKey(provider, apiKey, baseUrl),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.providerKeys(provider) });
    },
  });
}
