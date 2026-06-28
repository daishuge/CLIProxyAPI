/**
 * Routing strategy API.
 *
 * Backend contract (internal/api/handlers/management/config_basic.go):
 *   GET /routing/strategy -> { "strategy": "round-robin" | "fill-first" | <raw> }
 *   PUT /routing/strategy   body: { "value": "round-robin" | "fill-first" }
 *
 * The backend normalizes aliases (rr/roundrobin -> round-robin, ff/fillfirst ->
 * fill-first) and rejects anything else with 400.
 */
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { managementApi } from "./client";
import { queryKeys } from "../query";

/** The two routing strategies the backend accepts. */
export const ROUTING_STRATEGIES = ["round-robin", "fill-first"] as const;
export type RoutingStrategy = (typeof ROUTING_STRATEGIES)[number];

interface RoutingStrategyResponse {
  strategy?: string;
}

/** Read the current routing strategy. */
export function fetchRoutingStrategy(signal?: AbortSignal): Promise<string> {
  return managementApi
    .get<RoutingStrategyResponse>("/routing/strategy", signal ? { signal } : undefined)
    .then((res) => (res.strategy ?? "").trim());
}

/** Write the routing strategy. */
export function putRoutingStrategy(value: RoutingStrategy): Promise<void> {
  return managementApi.put<void>("/routing/strategy", { body: { value } });
}

/** Query hook for the routing strategy. */
export function useRoutingStrategyQuery(enabled = true) {
  return useQuery({
    queryKey: queryKeys.routingStrategy,
    queryFn: ({ signal }) => fetchRoutingStrategy(signal),
    enabled,
  });
}

/** Mutation hook to update the routing strategy. */
export function useUpdateRoutingStrategyMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (value: RoutingStrategy) => putRoutingStrategy(value),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.routingStrategy });
    },
  });
}
