/**
 * Conversation Logs API wrapper.
 *
 * PPAP-only surface. The three routes mount under the management prefix
 * (see internal/api/server.go around line 857):
 *   GET /v0/management/conversation-logs
 *   GET /v0/management/conversation-logs/tail
 *   GET /v0/management/conversation-logs/:id
 *
 * The backend response envelope always carries an `enabled` flag so the
 * UI can distinguish "no logs yet" from "conversation logging disabled".
 */

import { apiClient } from './client';

/** Lightweight summary shape returned by list + tail routes. */
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

/** Full entry shape returned by GET /conversation-logs/:id. */
export interface ConversationLogPayload {
  body?: unknown;
  text?: string;
  chunks?: string[];
  bytes?: number;
  truncated?: boolean;
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

export interface ConversationLogListResponse {
  enabled: boolean;
  entries: ConversationLogSummary[];
  next_cursor: string;
  malformed: number;
  tail: boolean;
}

export interface ConversationLogEntryResponse {
  enabled: boolean;
  entry: ConversationLogEntry;
}

export interface ConversationLogListQuery {
  limit?: number;
  cursor?: string;
  provider?: string;
  model?: string;
  path?: string;
  status_code?: number;
  has_error?: boolean;
  from?: string;
  to?: string;
  request_id?: string;
}

const stripUndefined = (input: ConversationLogListQuery): Record<string, string | number | boolean> => {
  const out: Record<string, string | number | boolean> = {};
  Object.entries(input).forEach(([key, value]) => {
    if (value === undefined || value === null || value === '') return;
    out[key] = value;
  });
  return out;
};

export const conversationLogsApi = {
  /** Paginated list; the backend accepts an opaque cursor. */
  list: (query: ConversationLogListQuery = {}) =>
    apiClient.get<ConversationLogListResponse>('/conversation-logs', {
      params: stripUndefined(query),
    }),

  /** Tail returns the freshest N summaries; cursor is ignored server-side. */
  tail: (query: Omit<ConversationLogListQuery, 'cursor'> = {}) =>
    apiClient.get<ConversationLogListResponse>('/conversation-logs/tail', {
      params: stripUndefined(query),
    }),

  /** Full-fidelity fetch of one entry, including truncated JSON bodies. */
  get: (id: string) =>
    apiClient.get<ConversationLogEntryResponse>(
      `/conversation-logs/${encodeURIComponent(id)}`
    ),
};
