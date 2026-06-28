/**
 * Management API response types. Field names mirror the Go backend JSON tags
 * (snake_case / kebab-case) exactly so responses deserialize without mapping.
 */

/** GET /healthz */
export interface HealthResponse {
  status: string;
}

/** GET /v0/management/latest-version */
export interface LatestVersionResponse {
  "latest-version": string;
}

/** Per-model usage rollup. */
export interface ModelSnapshot {
  total_requests: number;
  total_tokens: number;
  total_input_tokens: number;
  total_cached_tokens: number;
  cache_hit_rate: number;
  average_latency_ms: number;
  average_first_byte_latency_ms: number;
  tps: number;
}

/** Per-API-key usage rollup. */
export interface ApiSnapshot {
  total_requests: number;
  total_tokens: number;
  total_input_tokens: number;
  total_cached_tokens: number;
  cache_hit_rate: number;
  average_latency_ms: number;
  average_first_byte_latency_ms: number;
  tps: number;
  models: Record<string, ModelSnapshot>;
}

/** Aggregate usage snapshot returned inside the usage-statistics envelope. */
export interface StatisticsSnapshot {
  total_requests: number;
  success_count: number;
  failure_count: number;
  total_tokens: number;
  total_input_tokens: number;
  total_cached_tokens: number;
  cache_hit_rate: number;
  average_latency_ms: number;
  average_first_byte_latency_ms: number;
  tps: number;
  apis: Record<string, ApiSnapshot> | null;
  requests_by_day: Record<string, number> | null;
  requests_by_hour: Record<string, number> | null;
  tokens_by_day: Record<string, number> | null;
  tokens_by_hour: Record<string, number> | null;
}

/** GET /v0/management/usage-statistics */
export interface UsageStatisticsResponse {
  usage: StatisticsSnapshot;
  failed_requests: number;
}

/** GET /v0/management/usage-statistics-enabled */
export interface UsageStatisticsEnabledResponse {
  "usage-statistics-enabled"?: boolean;
  enabled?: boolean;
}

/** GET /v0/management/config — partial typing of the fields the panel reads. */
export interface ManagementConfig {
  debug?: boolean;
  "proxy-url"?: string;
  "request-log"?: boolean;
  "routing-strategy"?: string;
  "usage-statistics-enabled"?: boolean;
  [key: string]: unknown;
}

/** GET /v0/management/api-keys */
export interface ApiKeysResponse {
  "api-keys"?: string[];
  [key: string]: unknown;
}
