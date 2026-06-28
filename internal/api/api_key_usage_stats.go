package api

import (
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/usage"
)

// installAPIKeyUsageStatsLookup wires the rich-usage aggregation store into the
// per-key budget enforcement seam exposed by api_key_controls.go. Once
// installed, the request-count, input/total-token, and estimated-cost budgets
// in withinAPIKeyBudget become enforceable; before this call (and whenever stats
// is nil) those budgets degrade to a permissive no-op while the enabled and
// model allow/deny controls still apply.
//
// The aggregation store keys per-API metrics by the downstream API key (see
// RequestStatistics.Record, which uses record.APIKey as the primary key), so the
// lookup resolves a key's usage by reading the matching APISnapshot entry.
func installAPIKeyUsageStatsLookup(stats *usage.RequestStatistics) {
	if stats == nil {
		SetAPIKeyUsageStatsLookup(nil)
		return
	}
	SetAPIKeyUsageStatsLookup(func(apiKey string) (apiKeyUsageStats, bool) {
		apiKey = strings.TrimSpace(apiKey)
		if apiKey == "" {
			return apiKeyUsageStats{}, false
		}
		snapshot := stats.Snapshot()
		apiSnapshot, ok := snapshot.APIs[apiKey]
		if !ok {
			return apiKeyUsageStats{}, false
		}
		out := apiKeyUsageStats{
			TotalRequests:    apiSnapshot.TotalRequests,
			TotalInputTokens: apiSnapshot.TotalInputTokens,
			TotalTokens:      apiSnapshot.TotalTokens,
			Models:           make(map[string]apiKeyModelUsage, len(apiSnapshot.Models)),
		}
		for model, modelSnapshot := range apiSnapshot.Models {
			details := make([]apiKeyTokenStats, 0, len(modelSnapshot.Details))
			for _, detail := range modelSnapshot.Details {
				details = append(details, apiKeyTokenStats{
					InputTokens:  detail.Tokens.InputTokens,
					CachedTokens: detail.Tokens.CachedTokens,
					OutputTokens: detail.Tokens.OutputTokens,
					TotalTokens:  detail.Tokens.TotalTokens,
				})
			}
			out.Models[model] = apiKeyModelUsage{Details: details}
		}
		return out, true
	})
}
