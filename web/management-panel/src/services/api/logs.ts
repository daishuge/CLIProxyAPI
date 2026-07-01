/**
 * Logs API
 */

import { apiClient } from './client';
import { LOGS_TIMEOUT_MS } from '@/utils/constants';

export interface LogsQuery {
  after?: number;
}

export interface LogsResponse {
  lines: string[];
  'line-count': number;
  'latest-timestamp': number;
}

export interface ErrorLogFile {
  name: string;
  size?: number;
  modified?: number;
}

export interface ErrorLogsResponse {
  files?: ErrorLogFile[];
}

export type LogDataTarget = 'application' | 'request' | 'error-request' | 'temporary' | 'all';

export interface LogStorageBucket {
  size: number;
  files: number;
}

export interface LogStorageResponse {
  'log-directory': string;
  'total-size': number;
  'total-files': number;
  application: LogStorageBucket;
  request: LogStorageBucket;
  'error-request': LogStorageBucket;
  temporary: LogStorageBucket;
}

export interface ConversationLogPayload {
  body?: unknown;
  text?: string;
  chunks?: string[];
  bytes?: number;
  truncated?: boolean;
}

export interface ConversationLogSummary {
  id: string;
  request_id?: string;
  created_at: string;
  method?: string;
  path?: string;
  provider?: string;
  model?: string;
  status_code?: number;
  has_error: boolean;
  file: string;
  line_bytes: number;
}

export interface ConversationLogEntry {
  id: string;
  request_id?: string;
  created_at: string;
  completed_at?: string;
  latency_ms?: number;
  method?: string;
  path?: string;
  provider?: string;
  model?: string;
  upstream_url?: string;
  status_code?: number;
  error?: string;
  request_headers?: Record<string, string[]>;
  response_headers?: Record<string, string[]>;
  request?: ConversationLogPayload;
  response?: ConversationLogPayload;
  usage?: unknown;
  metadata?: Record<string, string>;
}

export interface ConversationLogsQuery {
  limit?: number;
  cursor?: string;
  request_id?: string;
  provider?: string;
  model?: string;
  path?: string;
  status_code?: number;
  has_error?: boolean;
  from?: string;
  to?: string;
}

export interface ConversationLogsResponse {
  enabled: boolean;
  entries: ConversationLogSummary[];
  next_cursor?: string;
  malformed: number;
  tail?: boolean;
}

export interface ConversationLogDetailResponse {
  enabled: boolean;
  entry: ConversationLogEntry;
}

export const logsApi = {
  fetchLogs: (params: LogsQuery = {}): Promise<LogsResponse> =>
    apiClient.get('/logs', { params, timeout: LOGS_TIMEOUT_MS }),

  fetchStorage: (): Promise<LogStorageResponse> =>
    apiClient.get('/logs/storage', { timeout: LOGS_TIMEOUT_MS }),

  clearLogs: (target: LogDataTarget = 'application') =>
    apiClient.delete('/logs', { params: { target } }),

  fetchErrorLogs: (): Promise<ErrorLogsResponse> =>
    apiClient.get('/request-error-logs', { timeout: LOGS_TIMEOUT_MS }),

  downloadErrorLog: (filename: string) =>
    apiClient.getRaw(`/request-error-logs/${encodeURIComponent(filename)}`, {
      responseType: 'blob',
      timeout: LOGS_TIMEOUT_MS,
    }),

  downloadRequestLogById: (id: string) =>
    apiClient.getRaw(`/request-log-by-id/${encodeURIComponent(id)}`, {
      responseType: 'blob',
      timeout: LOGS_TIMEOUT_MS,
    }),

  fetchConversationLogs: (params: ConversationLogsQuery = {}): Promise<ConversationLogsResponse> =>
    apiClient.get('/conversation-logs', { params, timeout: LOGS_TIMEOUT_MS }),

  fetchConversationLogTail: (
    params: Omit<ConversationLogsQuery, 'cursor'> = {}
  ): Promise<ConversationLogsResponse> =>
    apiClient.get('/conversation-logs/tail', { params, timeout: LOGS_TIMEOUT_MS }),

  fetchConversationLogDetail: (id: string): Promise<ConversationLogDetailResponse> =>
    apiClient.get(`/conversation-logs/${encodeURIComponent(id)}`, { timeout: LOGS_TIMEOUT_MS }),
};
