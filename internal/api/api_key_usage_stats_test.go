package api

import (
	"context"
	"testing"
	"time"

	proxyconfig "github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/usage"
	coreusage "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/usage"
)

// TestInstallAPIKeyUsageStatsLookupConnectsBudgets proves the Step6 bridge wires
// a real usage.RequestStatistics into the per-key budget seam so that the
// request-count, input/total-token, and estimated-cost budgets become
// enforceable end-to-end (Step3 only enforced model/enabled controls while the
// lookup was nil).
func TestInstallAPIKeyUsageStatsLookupConnectsBudgets(t *testing.T) {
	usage.SetStatisticsEnabled(true)
	stats := usage.NewRequestStatistics()
	// Record two gpt-5 requests for the key. gpt-5 input price is $1.25/M and
	// output $10/M, so 2 requests of 1000 input + 1000 output tokens cost about
	// 2 * (1000*1.25 + 1000*10) / 1_000_000 = $0.0225.
	for i := 0; i < 2; i++ {
		stats.Record(context.Background(), coreusage.Record{
			APIKey:      "budget-key",
			Model:       "gpt-5",
			RequestedAt: time.Date(2026, 5, 2, 12, i, 0, 0, time.UTC),
			Detail: coreusage.Detail{
				InputTokens:  1000,
				OutputTokens: 1000,
				TotalTokens:  2000,
			},
		})
	}

	installAPIKeyUsageStatsLookup(stats)
	t.Cleanup(func() { SetAPIKeyUsageStatsLookup(nil) })

	// Request-count budget: 2 recorded >= MaxRequests 2 blocks; 3 allows.
	if withinAPIKeyBudget(&proxyconfig.APIKeyControl{APIKey: "budget-key", MaxRequests: 2}) {
		t.Fatal("request budget should block at the recorded count")
	}
	if !withinAPIKeyBudget(&proxyconfig.APIKeyControl{APIKey: "budget-key", MaxRequests: 3}) {
		t.Fatal("request budget should allow below the recorded count")
	}

	// Total-token budget: 4000 total recorded.
	if withinAPIKeyBudget(&proxyconfig.APIKeyControl{APIKey: "budget-key", MaxTotalTokens: 4000}) {
		t.Fatal("total-token budget should block at the recorded total")
	}
	if !withinAPIKeyBudget(&proxyconfig.APIKeyControl{APIKey: "budget-key", MaxTotalTokens: 5000}) {
		t.Fatal("total-token budget should allow below the recorded total")
	}

	// Input-token budget: 2000 input recorded.
	if withinAPIKeyBudget(&proxyconfig.APIKeyControl{APIKey: "budget-key", MaxInputTokens: 2000}) {
		t.Fatal("input-token budget should block at the recorded input total")
	}

	// Estimated-cost budget: ~$0.0225 spent.
	if withinAPIKeyBudget(&proxyconfig.APIKeyControl{APIKey: "budget-key", MaxCostUSD: 0.02}) {
		t.Fatal("cost budget should block once estimated spend exceeds the cap")
	}
	if !withinAPIKeyBudget(&proxyconfig.APIKeyControl{APIKey: "budget-key", MaxCostUSD: 1}) {
		t.Fatal("cost budget should allow below the cap")
	}

	// Unknown key returns no stats, so budgets stay permissive.
	if !withinAPIKeyBudget(&proxyconfig.APIKeyControl{APIKey: "other-key", MaxRequests: 1}) {
		t.Fatal("unknown key should be permissive")
	}
}

func TestInstallAPIKeyUsageStatsLookupNilDisables(t *testing.T) {
	installAPIKeyUsageStatsLookup(nil)
	t.Cleanup(func() { SetAPIKeyUsageStatsLookup(nil) })
	if !withinAPIKeyBudget(&proxyconfig.APIKeyControl{APIKey: "k", MaxRequests: 1}) {
		t.Fatal("nil provider must degrade budgets to permissive")
	}
}
