/**
 * Logs API — hooks and fetch helpers for request logs, error logs,
 * conversation logs and log storage management.
 */
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { managementApi } from "./client";
import { queryKeys } from "../query";

// ===== Types =====

/** A single request log entry. */
export interface LogEntry {
  timestamp: string;
  method: string;
  path: string;
  status: number;
  latency_ms: number;
  model?: string;
  api_key?: string;
  [key: string]: unknown;
}

/** GET /request-log may return the log array or a boolean on/off flag. */
export type RequestLogResponse =
  | { "request-log": LogEntry[] }
  | { "request-log": boolean };

/** A single error log entry. */
export interface ErrorLogEntry {
  timestamp: string;
  method: string;
  path: string;
  status: number;
  error: string;
  [key: string]: unknown;
}

interface ErrorLogsResponse {
  "request-error-logs"?: ErrorLogEntry[];
}

/** A single conversation log entry (PPAP private feature). */
export interface ConversationLogEntry {
  id: string;
  api_key: string;
  model: string;
  timestamp: string;
  size_bytes: number;
}

interface ConversationLogsResponse {
  entries?: ConversationLogEntry[];
}

/** Log storage info. */
export interface LogStorageInfo {
  total_size_bytes: number;
  file_count: number;
  [key: string]: unknown;
}

// ===== Fetch functions =====

/** Fetch request logs (may be an array or boolean). */
export function fetchRequestLogs(signal?: AbortSignal): Promise<RequestLogResponse> {
  return managementApi.get<RequestLogResponse>("/request-log", signal ? { signal } : undefined);
}

/** Fetch error logs. */
export function fetchErrorLogs(signal?: AbortSignal): Promise<ErrorLogEntry[]> {
  return managementApi
    .get<ErrorLogsResponse>("/request-error-logs", signal ? { signal } : undefined)
    .then((res) => res["request-error-logs"] ?? []);
}

/** Fetch conversation log listing. */
export function fetchConversationLogs(signal?: AbortSignal): Promise<ConversationLogEntry[]> {
  return managementApi
    .get<ConversationLogsResponse>("/conversation-logs", signal ? { signal } : undefined)
    .then((res) => res.entries ?? []);
}

/** Tail a conversation log. */
export function fetchConversationLogTail(
  id: string,
  lines = 100,
  signal?: AbortSignal,
): Promise<string> {
  return managementApi.get<string>("/conversation-logs/tail", {
    query: { id, lines },
    ...(signal ? { signal } : {}),
  });
}

/** Fetch log storage info. */
export function fetchLogStorage(signal?: AbortSignal): Promise<LogStorageInfo> {
  return managementApi.get<LogStorageInfo>("/logs/storage", signal ? { signal } : undefined);
}

/** Delete logs of a given type. */
export function deleteLogs(type: string): Promise<void> {
  return managementApi.delete<void>("/logs", { query: { type } });
}

// ===== React Query hooks =====

export function useRequestLogsQuery(enabled = true) {
  return useQuery({
    queryKey: queryKeys.requestLogs,
    queryFn: ({ signal }) => fetchRequestLogs(signal),
    enabled,
  });
}

export function useErrorLogsQuery(enabled = true) {
  return useQuery({
    queryKey: queryKeys.errorLogs,
    queryFn: ({ signal }) => fetchErrorLogs(signal),
    enabled,
  });
}

export function useConversationLogsQuery(enabled = true) {
  return useQuery({
    queryKey: queryKeys.conversationLogs,
    queryFn: ({ signal }) => fetchConversationLogs(signal),
    enabled,
  });
}

export function useConversationLogDetailQuery(id: string | null) {
  return useQuery({
    queryKey: queryKeys.conversationLogDetail(id ?? ""),
    queryFn: ({ signal }) => fetchConversationLogTail(id!, 200, signal),
    enabled: id !== null,
  });
}

export function useLogStorageQuery(enabled = true) {
  return useQuery({
    queryKey: queryKeys.logStorage,
    queryFn: ({ signal }) => fetchLogStorage(signal),
    enabled,
  });
}

export function useDeleteLogsMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (type: string) => deleteLogs(type),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.requestLogs });
      void queryClient.invalidateQueries({ queryKey: queryKeys.errorLogs });
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversationLogs });
      void queryClient.invalidateQueries({ queryKey: queryKeys.logStorage });
    },
  });
}
