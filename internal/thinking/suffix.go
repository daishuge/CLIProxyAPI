// Package thinking provides unified thinking configuration processing.
//
// This file implements suffix parsing functionality for extracting
// thinking configuration from model names in the format model(value).
package thinking

import (
	"strconv"
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/registry"
)

// ParseSuffix extracts thinking suffix from a model name.
//
// The suffix format is: model-name(value)
// Examples:
//   - "claude-sonnet-4-5(16384)" -> ModelName="claude-sonnet-4-5", RawSuffix="16384"
//   - "gpt-5.2(high)" -> ModelName="gpt-5.2", RawSuffix="high"
//   - "gemini-2.5-pro" -> ModelName="gemini-2.5-pro", HasSuffix=false
//
// This function only extracts the suffix; it does not validate or interpret
// the suffix content. Use ParseNumericSuffix, ParseLevelSuffix, etc. for
// content interpretation.
func ParseSuffix(model string) SuffixResult {
	// Find the last opening parenthesis
	lastOpen := strings.LastIndex(model, "(")
	if lastOpen == -1 {
		return SuffixResult{ModelName: model, HasSuffix: false}
	}

	// Check if the string ends with a closing parenthesis
	if !strings.HasSuffix(model, ")") {
		return SuffixResult{ModelName: model, HasSuffix: false}
	}

	// Extract components
	modelName := model[:lastOpen]
	rawSuffix := model[lastOpen+1 : len(model)-1]

	return SuffixResult{
		ModelName: modelName,
		HasSuffix: true,
		RawSuffix: rawSuffix,
	}
}

var hyphenLevelSuffixes = map[string]ThinkingLevel{
	"low":    LevelLow,
	"medium": LevelMedium,
	"high":   LevelHigh,
	"xhigh":  LevelXHigh,
}

// ParseSuffixAllowHyphen extracts model(level) and model-level suffixes.
//
// Hyphen suffix parsing is intentionally limited to level aliases used as model
// convenience aliases. Numeric budgets remain parenthesis-only so ordinary
// hyphenated model IDs are not treated as budget syntax.
func ParseSuffixAllowHyphen(model string) SuffixResult {
	if result := ParseSuffix(model); result.HasSuffix {
		return result
	}
	return ParseHyphenLevelSuffix(model)
}

// ParseHyphenLevelSuffix extracts a model-low/model-medium/model-high/model-xhigh suffix.
// It does not validate that the base model supports the level.
func ParseHyphenLevelSuffix(model string) SuffixResult {
	model = strings.TrimSpace(model)
	lastDash := strings.LastIndex(model, "-")
	if lastDash <= 0 || lastDash == len(model)-1 {
		return SuffixResult{ModelName: model, HasSuffix: false}
	}
	base := strings.TrimSpace(model[:lastDash])
	raw := strings.TrimSpace(model[lastDash+1:])
	if base == "" || raw == "" {
		return SuffixResult{ModelName: model, HasSuffix: false}
	}
	if _, ok := hyphenLevelSuffixes[strings.ToLower(raw)]; !ok {
		return SuffixResult{ModelName: model, HasSuffix: false}
	}
	return SuffixResult{ModelName: base, HasSuffix: true, RawSuffix: strings.ToLower(raw)}
}

// ParseSuffixForModel extracts a thinking suffix only when it is safe to do so.
// Parenthesized suffixes are always honored for backward compatibility. Hyphen
// level aliases are honored only when the exact model is not registered and the
// stripped base model exists with the requested thinking level.
func ParseSuffixForModel(model string, provider ...string) SuffixResult {
	model = strings.TrimSpace(model)
	if model == "" {
		return SuffixResult{ModelName: model, HasSuffix: false}
	}
	if result := ParseSuffix(model); result.HasSuffix {
		return result
	}
	hyphenResult := ParseHyphenLevelSuffix(model)
	exactInfo := registry.LookupModelInfo(model, firstProvider(provider...))
	if exactInfo != nil {
		if exactInfo.ThinkingAliasBase == "" || !hyphenResult.HasSuffix {
			return SuffixResult{ModelName: model, HasSuffix: false}
		}
		aliasBase := strings.TrimSpace(exactInfo.ThinkingAliasBase)
		if aliasBase == "" || !strings.EqualFold(aliasBase, hyphenResult.ModelName) {
			return SuffixResult{ModelName: model, HasSuffix: false}
		}
		if modelSupportsLevel(exactInfo, hyphenResult.RawSuffix) {
			return hyphenResult
		}
		return SuffixResult{ModelName: model, HasSuffix: false}
	}
	if !hyphenResult.HasSuffix {
		return hyphenResult
	}
	info := registry.LookupModelInfo(hyphenResult.ModelName, firstProvider(provider...))
	if !modelSupportsLevel(info, hyphenResult.RawSuffix) {
		return SuffixResult{ModelName: model, HasSuffix: false}
	}
	return hyphenResult
}

// FormatSuffix appends the parsed suffix using the canonical parenthesized form.
func FormatSuffix(modelName string, suffix SuffixResult) string {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" || !suffix.HasSuffix || strings.TrimSpace(suffix.RawSuffix) == "" {
		return modelName
	}
	return modelName + "(" + strings.TrimSpace(suffix.RawSuffix) + ")"
}

func firstProvider(provider ...string) string {
	if len(provider) == 0 {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(provider[0]))
}

func modelSupportsLevel(info *registry.ModelInfo, rawLevel string) bool {
	if info == nil || info.Thinking == nil {
		return false
	}
	rawLevel = strings.ToLower(strings.TrimSpace(rawLevel))
	if rawLevel == "" {
		return false
	}
	for _, level := range info.Thinking.Levels {
		if strings.EqualFold(strings.TrimSpace(level), rawLevel) {
			return true
		}
	}
	return false
}

// ParseNumericSuffix attempts to parse a raw suffix as a numeric budget value.
//
// This function parses the raw suffix content (from ParseSuffix.RawSuffix) as an integer.
// Only non-negative integers are considered valid numeric suffixes.
//
// Platform note: The budget value uses Go's int type, which is 32-bit on 32-bit
// systems and 64-bit on 64-bit systems. Values exceeding the platform's int range
// will return ok=false.
//
// Leading zeros are accepted: "08192" parses as 8192.
//
// Examples:
//   - "8192" -> budget=8192, ok=true
//   - "0" -> budget=0, ok=true (represents ModeNone)
//   - "08192" -> budget=8192, ok=true (leading zeros accepted)
//   - "-1" -> budget=0, ok=false (negative numbers are not valid numeric suffixes)
//   - "high" -> budget=0, ok=false (not a number)
//   - "9223372036854775808" -> budget=0, ok=false (overflow on 64-bit systems)
//
// For special handling of -1 as auto mode, use ParseSpecialSuffix instead.
func ParseNumericSuffix(rawSuffix string) (budget int, ok bool) {
	if rawSuffix == "" {
		return 0, false
	}

	value, err := strconv.Atoi(rawSuffix)
	if err != nil {
		return 0, false
	}

	// Negative numbers are not valid numeric suffixes
	// -1 should be handled by special value parsing as "auto"
	if value < 0 {
		return 0, false
	}

	return value, true
}

// ParseSpecialSuffix attempts to parse a raw suffix as a special thinking mode value.
//
// This function handles special strings that represent a change in thinking mode:
//   - "none" -> ModeNone (disables thinking)
//   - "auto" -> ModeAuto (automatic/dynamic thinking)
//   - "-1"   -> ModeAuto (numeric representation of auto mode)
//
// String values are case-insensitive.
func ParseSpecialSuffix(rawSuffix string) (mode ThinkingMode, ok bool) {
	if rawSuffix == "" {
		return ModeBudget, false
	}

	// Case-insensitive matching
	switch strings.ToLower(rawSuffix) {
	case "none":
		return ModeNone, true
	case "auto", "-1":
		return ModeAuto, true
	default:
		return ModeBudget, false
	}
}

// ParseLevelSuffix attempts to parse a raw suffix as a discrete thinking level.
//
// This function parses the raw suffix content (from ParseSuffix.RawSuffix) as a level.
// Only discrete effort levels are valid: minimal, low, medium, high, xhigh, max.
// Level matching is case-insensitive.
//
// Special values (none, auto) are NOT handled by this function; use ParseSpecialSuffix
// instead. This separation allows callers to prioritize special value handling.
//
// Examples:
//   - "high" -> level=LevelHigh, ok=true
//   - "HIGH" -> level=LevelHigh, ok=true (case insensitive)
//   - "medium" -> level=LevelMedium, ok=true
//   - "none" -> level="", ok=false (special value, use ParseSpecialSuffix)
//   - "auto" -> level="", ok=false (special value, use ParseSpecialSuffix)
//   - "8192" -> level="", ok=false (numeric, use ParseNumericSuffix)
//   - "ultra" -> level="", ok=false (unknown level)
func ParseLevelSuffix(rawSuffix string) (level ThinkingLevel, ok bool) {
	if rawSuffix == "" {
		return "", false
	}

	// Case-insensitive matching
	switch strings.ToLower(rawSuffix) {
	case "minimal":
		return LevelMinimal, true
	case "low":
		return LevelLow, true
	case "medium":
		return LevelMedium, true
	case "high":
		return LevelHigh, true
	case "xhigh":
		return LevelXHigh, true
	case "max":
		return LevelMax, true
	default:
		return "", false
	}
}
