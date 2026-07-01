import { apiClient } from './client';
import type {
  UsageExportPayload,
  UsageImportResult,
  UsageStatisticsResponse,
} from '@/types';

const USAGE_TIMEOUT_MS = 20 * 1000;

export const usageApi = {
  getStatistics: () =>
    apiClient.get<UsageStatisticsResponse>('/usage', {
      timeout: USAGE_TIMEOUT_MS,
    }),

  exportStatistics: () =>
    apiClient.get<UsageExportPayload>('/usage/export', {
      timeout: USAGE_TIMEOUT_MS,
    }),

  importStatistics: (payload: UsageExportPayload) =>
    apiClient.post<UsageImportResult>('/usage/import', payload, {
      timeout: USAGE_TIMEOUT_MS,
    }),
};
