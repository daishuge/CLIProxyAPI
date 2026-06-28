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
};
