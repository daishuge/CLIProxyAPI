import { QueryClient } from "@tanstack/react-query";
import { ApiError } from "./api";

/**
 * Shared React Query client. Auth failures (401) are never retried — the client
 * interceptor already triggers logout — and other errors get a single retry.
 */
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: (failureCount, error) => {
        if (error instanceof ApiError && (error.unauthorized || error.kind === "forbidden")) {
          return false;
        }
        return failureCount < 1;
      },
      staleTime: 30_000,
      refetchOnWindowFocus: false,
    },
    mutations: {
      retry: false,
    },
  },
});

/** Centralized query-key registry to keep cache invalidation consistent. */
export const queryKeys = {
  health: ["health"] as const,
  config: ["config"] as const,
  usageStatistics: ["usage-statistics"] as const,
  latestVersion: ["latest-version"] as const,
  apiKeys: ["api-keys"] as const,
  // Providers & Auth
  authFiles: ["auth-files"] as const,
  authFileModels: (name: string) => ["auth-files", "models", name] as const,
  providerKeys: (provider: string) => ["provider-keys", provider] as const,
  // Routing & Models
  modelDefinitions: (channel: string) => ["model-definitions", channel] as const,
  modelAliases: ["oauth-model-alias"] as const,
  excludedModels: ["oauth-excluded-models"] as const,
  forceModelPrefix: ["force-model-prefix"] as const,
  customUpstreams: ["custom-upstreams"] as const,
  routingStrategy: ["routing", "strategy"] as const,
  // Traffic & Usage
  apiKeyUsage: ["api-key-usage"] as const,
  // Logs
  requestLogs: ["request-logs"] as const,
  errorLogs: ["error-logs"] as const,
  conversationLogs: ["conversation-logs"] as const,
  conversationLogDetail: (id: string) => ["conversation-logs", id] as const,
  logStorage: ["log-storage"] as const,
};
