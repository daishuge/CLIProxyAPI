// Package compat provides Codex protocol compatibility helpers that bridge
// client-facing request shapes to the values currently accepted by the Codex
// upstream. These helpers are deliberately free of provider routing or
// executor state so they can be reused across executors and handlers.
package compat

import (
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// NormalizeServiceTier rewrites client-facing Codex service tiers to the wire value
// currently accepted by the Codex upstream. Unknown values are removed.
func NormalizeServiceTier(rawJSON []byte) []byte {
	tierResult := gjson.GetBytes(rawJSON, "service_tier")
	if !tierResult.Exists() {
		return rawJSON
	}
	tier, ok := NormalizeServiceTierValue(tierResult.String())
	if !ok {
		rawJSON, _ = sjson.DeleteBytes(rawJSON, "service_tier")
		return rawJSON
	}
	rawJSON, _ = sjson.SetBytes(rawJSON, "service_tier", tier)
	return rawJSON
}

// NormalizeServiceTierValue returns the Codex wire value for a downstream service tier.
func NormalizeServiceTierValue(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "priority", "fast":
		return "priority", true
	default:
		return "", false
	}
}

// NormalizeFastModelAlias strips the client-facing fast suffix from a model
// alias while preserving any thinking-strength suffix before it. Provider
// routing decides whether the stripped model is a Codex model.
func NormalizeFastModelAlias(model string) (string, bool) {
	model = strings.TrimSpace(model)
	if model == "" {
		return "", false
	}
	if !strings.HasSuffix(strings.ToLower(model), "-fast") {
		return model, false
	}
	base := strings.TrimSpace(model[:len(model)-len("-fast")])
	if base == "" {
		return model, false
	}
	return base, true
}

// RequestWantsPriorityServiceTier reports whether the client requested Codex
// priority service either by request parameter or by a fast model alias.
func RequestWantsPriorityServiceTier(originalJSON []byte, model string) bool {
	if _, ok := NormalizeFastModelAlias(model); ok {
		return true
	}
	if len(originalJSON) == 0 {
		return false
	}
	if rawModel := gjson.GetBytes(originalJSON, "model").String(); rawModel != "" {
		if _, ok := NormalizeFastModelAlias(rawModel); ok {
			return true
		}
	}
	serviceTier := gjson.GetBytes(originalJSON, "service_tier")
	if !serviceTier.Exists() {
		return false
	}
	tier, ok := NormalizeServiceTierValue(serviceTier.String())
	return ok && tier == "priority"
}

// ApplyRequestedServiceTier normalizes the body service tier and applies the
// priority tier requested by the original client payload or model alias.
func ApplyRequestedServiceTier(body, originalJSON []byte, model string) []byte {
	body = NormalizeServiceTier(body)
	if RequestWantsPriorityServiceTier(originalJSON, model) {
		body, _ = sjson.SetBytes(body, "service_tier", "priority")
	}
	return body
}
