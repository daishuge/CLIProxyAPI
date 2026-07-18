/**
 * Structured CRUD for `api-key-controls` — the per-downstream-key limits and
 * whitelists that PPAP's fork extended (name, model whitelist/blacklist,
 * budget caps, preset prompt). The backend counterpart is
 * internal/api/handlers/management/api_key_controls.go.
 *
 * Every response merges live usage + estimated USD spend so a single request
 * powers both the list view and per-key detail drawer.
 */

import { apiClient } from './client';

export interface APIKeyControlModelBreakdown {
  name: string;
  requests: number;
  tokens: number;
  cached_tokens: number;
  used_usd: number;
  avg_latency_ms: number;
  price_source:
    | 'external:exact'
    | 'external:pattern'
    | 'builtin:gpt'
    | 'builtin:gpt-fallback'
    | 'unknown';
  price_matched?: string;
  price_input_per_m: number;
  price_cached_input_per_m: number;
  price_output_per_m: number;
}

export interface APIKeyControlRecentRequest {
  timestamp: string;
  model: string;
  source?: string;
  input_tokens: number;
  cached_tokens: number;
  output_tokens: number;
  total_tokens: number;
  latency_ms: number;
  failed: boolean;
  cost_usd: number;
}

export interface APIKeyControlUsage {
  total_requests: number;
  failure_count: number;
  total_tokens: number;
  total_input_tokens: number;
  total_cached_tokens: number;
  used_usd: number;
  remaining_usd: number;
  used_percent: number;
  exhausted: boolean;
  avg_latency_ms: number;
  models: APIKeyControlModelBreakdown[];
  recent?: APIKeyControlRecentRequest[];
}

export interface APIKeyControl {
  index: number;
  name: string;
  'api-key': string;
  'api-key-mask': string;
  'api-key-hash': string;
  enabled?: boolean | null;
  unlimited: boolean;
  models?: string[];
  'excluded-models'?: string[];
  'max-requests': number;
  'max-input-tokens': number;
  'max-total-tokens': number;
  'max-cost-usd': number;
  'preset-prompt-enabled': boolean;
  'preset-prompt-excerpt'?: string;
  usage?: APIKeyControlUsage;
}

export interface ListResponse {
  'api-key-controls': APIKeyControl[];
  external_pricing_file: string;
}

export interface CreateBody {
  name: string;
  'api-key'?: string;
  models?: string[];
  'excluded-models'?: string[];
  'max-cost-usd'?: number;
  'max-requests'?: number;
  'max-input-tokens'?: number;
  'max-total-tokens'?: number;
  enabled?: boolean;
  unlimited?: boolean;
}

export interface PatchBody {
  target_name?: string;
  target_index?: number;
  value: Partial<{
    name: string;
    'api-key': string;
    enabled: boolean;
    unlimited: boolean;
    models: string[];
    'excluded-models': string[];
    'max-requests': number;
    'max-input-tokens': number;
    'max-total-tokens': number;
    'max-cost-usd': number;
    'preset-prompt-enabled': boolean;
  }>;
}

export const apiKeyControlsApi = {
  list: (opts?: { recent?: number; mask_key?: boolean }) => {
    const params = new URLSearchParams();
    if (opts?.recent !== undefined) params.set('recent', String(opts.recent));
    if (opts?.mask_key) params.set('mask_key', '1');
    const qs = params.toString();
    return apiClient.get<ListResponse>(`/api-key-controls${qs ? `?${qs}` : ''}`);
  },
  create: (body: CreateBody) =>
    apiClient.post<{ created: APIKeyControl }>('/api-key-controls', body),
  patch: (body: PatchBody) =>
    apiClient.patch<{ updated: APIKeyControl }>('/api-key-controls', body),
  remove: (name: string, keepAPIKey = false) =>
    apiClient.delete('/api-key-controls', {
      data: { name, keep_api_key: keepAPIKey },
    }),
};
