package registry

import "strings"

// localCatalogOverrides lists model entries this fork adds on top of whatever
// the models catalog (embedded internal/registry/models/models.json, or the
// remote refresh from router-for-me/models — see model_updater.go) currently
// provides. That catalog is fully REPLACED on every successful load, embedded
// or remote, so a plain edit to the embedded JSON alone would be silently
// wiped by the next periodic remote refresh (every 3h) as long as upstream
// hasn't added the model yet.
//
// applyLocalCatalogOverrides re-applies these entries after every load
// (see loadModelsFromBytes in model_updater.go), so they survive both the
// startup embed load and every subsequent remote refresh. Each entry is only
// appended when missing, so once upstream's catalog includes the same model
// ID this becomes a harmless no-op — remove the entry here once that happens.
var localCatalogOverrides = []struct {
	section string
	model   *ModelInfo
}{
	{
		// Anthropic released Claude Sonnet 5 (2026-06-30); as of this writing
		// upstream router-for-me/models has not added it yet (only 4.x
		// Sonnet variants). Verified via a direct Anthropic-native call
		// through this fork's CPA OAuth credential that "claude-sonnet-5" is
		// a valid, working upstream model ID before adding it here.
		section: "claude",
		model: &ModelInfo{
			ID:                  "claude-sonnet-5",
			Object:              "model",
			Created:             1782877679,
			OwnedBy:             "anthropic",
			Type:                "claude",
			DisplayName:         "Claude Sonnet 5",
			ContextLength:       200000,
			MaxCompletionTokens: 64000,
			Thinking: &ThinkingSupport{
				Min:         1024,
				Max:         128000,
				ZeroAllowed: true,
				Levels:      []string{"low", "medium", "high", "max"},
			},
		},
	},
}

func applyLocalCatalogOverrides(parsed *staticModelsJSON) {
	if parsed == nil {
		return
	}
	for _, ov := range localCatalogOverrides {
		if ov.model == nil || strings.TrimSpace(ov.model.ID) == "" {
			continue
		}
		var target *[]*ModelInfo
		switch ov.section {
		case "claude":
			target = &parsed.Claude
		default:
			continue
		}
		exists := false
		for _, m := range *target {
			if m != nil && strings.EqualFold(strings.TrimSpace(m.ID), ov.model.ID) {
				exists = true
				break
			}
		}
		if !exists {
			*target = append(*target, ov.model)
		}
	}
}
