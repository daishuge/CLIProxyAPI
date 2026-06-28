package compat

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestNormalizeFastModelAlias(t *testing.T) {
	tests := []struct {
		name      string
		model     string
		wantModel string
		wantFast  bool
	}{
		{name: "plain", model: "gpt-5.5", wantModel: "gpt-5.5", wantFast: false},
		{name: "fast", model: "gpt-5.5-fast", wantModel: "gpt-5.5", wantFast: true},
		{name: "thinking fast", model: "gpt-5.5-high-fast", wantModel: "gpt-5.5-high", wantFast: true},
		{name: "spark fast", model: "gpt-5.3-codex-spark-fast", wantModel: "gpt-5.3-codex-spark", wantFast: true},
		{name: "spark thinking fast", model: "gpt-5.3-codex-spark-high-fast", wantModel: "gpt-5.3-codex-spark-high", wantFast: true},
		{name: "case insensitive suffix", model: "gpt-5.5-FAST", wantModel: "gpt-5.5", wantFast: true},
		{name: "generic suffix is stripped before provider routing", model: "imagen-4.0-fast", wantModel: "imagen-4.0", wantFast: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotModel, gotFast := NormalizeFastModelAlias(tt.model)
			if gotModel != tt.wantModel || gotFast != tt.wantFast {
				t.Fatalf("NormalizeFastModelAlias(%q) = (%q, %v), want (%q, %v)", tt.model, gotModel, gotFast, tt.wantModel, tt.wantFast)
			}
		})
	}
}

func TestApplyRequestedServiceTier(t *testing.T) {
	body := []byte(`{"model":"gpt-5.5"}`)
	out := ApplyRequestedServiceTier(body, []byte(`{"model":"gpt-5.5-fast"}`), "gpt-5.5")
	if got := gjson.GetBytes(out, "service_tier").String(); got != "priority" {
		t.Fatalf("service_tier = %q, want priority. Output: %s", got, string(out))
	}

	out = ApplyRequestedServiceTier([]byte(`{"service_tier":"fast"}`), []byte(`{"model":"gpt-5.5"}`), "gpt-5.5")
	if got := gjson.GetBytes(out, "service_tier").String(); got != "priority" {
		t.Fatalf("explicit service_tier = %q, want priority. Output: %s", got, string(out))
	}

	out = ApplyRequestedServiceTier([]byte(`{"service_tier":"default"}`), []byte(`{"model":"gpt-5.5"}`), "gpt-5.5")
	if gjson.GetBytes(out, "service_tier").Exists() {
		t.Fatalf("service_tier should be removed. Output: %s", string(out))
	}
}
