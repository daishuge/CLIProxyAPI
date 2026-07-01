export interface UsageTokenStats {
  input_tokens: number;
  output_tokens: number;
  reasoning_tokens: number;
  cached_tokens: number;
  total_tokens: number;
}

export interface UsageRequestDetail {
  timestamp: string;
  latency_ms: number;
  first_byte_latency_ms: number;
  source: string;
  auth_index: string;
  tokens: UsageTokenStats;
  failed: boolean;
}

export interface UsageModelSnapshot {
  total_requests: number;
  total_tokens: number;
  total_input_tokens: number;
  total_cached_tokens: number;
  cache_hit_rate: number;
  average_latency_ms: number;
  average_first_byte_latency_ms: number;
  tps: number;
  details?: UsageRequestDetail[];
}

export interface UsageApiSnapshot {
  total_requests: number;
  total_tokens: number;
  total_input_tokens: number;
  total_cached_tokens: number;
  cache_hit_rate: number;
  average_latency_ms: number;
  average_first_byte_latency_ms: number;
  tps: number;
  models?: Record<string, UsageModelSnapshot>;
}

export interface UsageStatisticsSnapshot {
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
  apis?: Record<string, UsageApiSnapshot>;
  requests_by_day?: Record<string, number>;
  requests_by_hour?: Record<string, number>;
  tokens_by_day?: Record<string, number>;
  tokens_by_hour?: Record<string, number>;
}

export interface UsageStatisticsResponse {
  usage: UsageStatisticsSnapshot;
  failed_requests: number;
}

export interface UsageExportPayload {
  version: number;
  exported_at: string;
  usage: UsageStatisticsSnapshot;
}

export interface UsageImportResult {
  added: number;
  skipped: number;
  total_requests: number;
  failed_requests: number;
}
