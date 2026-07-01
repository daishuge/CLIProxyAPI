package watcher

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
)

// TestSnapshotCoreAuths_FourDistinctFilesAllSurvive reproduces the live Singapore
// auth set (claude OAuth, two codex OAuth, one gemini Cloud-Code OAuth) as four
// distinct auth files and asserts that ALL FOUR survive synthesis + the
// dispatcher dedup (prepareAuthUpdatesLocked keys by auth.ID; distinct files must
// not collide). This guards against the canary "4 -> 3 auth entries" drop being a
// source bug: the type=gemini file must be preserved and normalized to the native
// gemini-cli provider, and the other three must each keep their own provider.
func TestSnapshotCoreAuths_FourDistinctFilesAllSurvive(t *testing.T) {
	authDir := t.TempDir()

	writeAuth := func(name string, meta map[string]any) {
		data, err := json.Marshal(meta)
		if err != nil {
			t.Fatalf("marshal %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(authDir, name), data, 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	// Four distinct OAuth account files, mirroring the live deployment.
	writeAuth("claude-yujin.json", map[string]any{
		"type": "claude", "email": "yujin@example.com",
		"access_token": "x", "refresh_token": "y",
	})
	writeAuth("codex-blmdxiao.json", map[string]any{
		"type": "codex", "email": "blmdxiao@example.com",
		"access_token": "x", "refresh_token": "y",
	})
	writeAuth("codex-huangxxiu.json", map[string]any{
		"type": "codex", "email": "huangxxiu@example.com",
		"access_token": "x", "refresh_token": "y",
	})
	writeAuth("gemini-walkerhuang.json", map[string]any{
		"type": "gemini", "email": "walkerhuang@example.com",
		"project_id": "proj-x", "access_token": "x", "refresh_token": "y",
		"scopes": []string{"https://www.googleapis.com/auth/cloud-platform"},
	})

	cfg := &config.Config{AuthDir: authDir} // NO config API keys (0 gemini keys), like live
	w := &Watcher{authDir: authDir}
	w.SetConfig(cfg)

	auths := w.SnapshotCoreAuths()
	if len(auths) != 4 {
		t.Fatalf("expected 4 synthesized auths, got %d", len(auths))
	}

	// The dispatcher dedup keys by auth.ID (prepareAuthUpdatesLocked builds
	// newState[auth.ID]); four distinct files must not collide down to three.
	byProvider := map[string]int{}
	seenIDs := map[string]bool{}
	for _, a := range auths {
		if a == nil {
			continue
		}
		if seenIDs[a.ID] {
			t.Fatalf("duplicate auth ID (would dedup to fewer entries): %q", a.ID)
		}
		seenIDs[a.ID] = true
		byProvider[a.Provider]++
	}
	if len(seenIDs) != 4 {
		t.Fatalf("expected 4 distinct auth IDs (no dedup collision), got %d (byProvider=%v)", len(seenIDs), byProvider)
	}
	if byProvider["claude"] != 1 {
		t.Fatalf("expected 1 claude auth, got %d", byProvider["claude"])
	}
	if byProvider["codex"] != 2 {
		t.Fatalf("expected 2 codex auths, got %d", byProvider["codex"])
	}
	// The type=gemini OAuth account must be preserved AS gemini-cli (native gemini
	// is API-key only; the OAuth account must reach the gemini-cli executor).
	if byProvider["gemini-cli"] != 1 {
		t.Fatalf("expected 1 gemini-cli auth (from the type=gemini OAuth file), got %d (byProvider=%v)", byProvider["gemini-cli"], byProvider)
	}
	if byProvider["gemini"] != 0 {
		t.Fatalf("did not expect any native gemini auth (OAuth must be gemini-cli), got %d", byProvider["gemini"])
	}
}
