package executor

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestStripDeprecatedSamplingParams(t *testing.T) {
	cases := []struct {
		name     string
		model    string
		body     string
		wantTemp bool // temperature should remain
		wantTopP bool // top_p should remain
	}{
		{"sonnet5 strips both", "claude-sonnet-5", `{"temperature":0.2,"top_p":0.9,"max_tokens":10}`, false, false},
		{"opus4-8 strips both", "claude-opus-4-8", `{"temperature":0.2,"top_p":0.9}`, false, false},
		{"opus4-8 effort variant strips", "claude-opus-4-8-high", `{"temperature":0.5}`, false, false},
		{"sonnet4-6 keeps temperature", "claude-sonnet-4-6", `{"temperature":0.2,"top_p":0.9}`, true, true},
		{"unrelated model keeps params", "gpt-5.5", `{"temperature":0.2,"top_p":0.9}`, true, true},
		{"sonnet5 dated variant strips", "claude-sonnet-5-20260930", `{"temperature":1}`, false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := stripDeprecatedSamplingParams([]byte(c.body), c.model)
			gotTemp := gjson.GetBytes(out, "temperature").Exists()
			gotTopP := gjson.GetBytes(out, "top_p").Exists()
			if gotTemp != c.wantTemp {
				t.Errorf("temperature present=%v, want %v (out=%s)", gotTemp, c.wantTemp, out)
			}
			if gotTopP != c.wantTopP {
				t.Errorf("top_p present=%v, want %v (out=%s)", gotTopP, c.wantTopP, out)
			}
		})
	}
}

func TestModelDeprecatesSamplingParams(t *testing.T) {
	deprecated := []string{"claude-sonnet-5", "claude-opus-4-8", "claude-opus-4-8-medium", "CLAUDE-SONNET-5"}
	ok := []string{"claude-sonnet-4-6", "claude-opus-4-6", "claude-haiku-4-5-20251001", "gpt-5.5", ""}
	for _, m := range deprecated {
		if !modelDeprecatesSamplingParams(m) {
			t.Errorf("expected %q to be deprecated", m)
		}
	}
	for _, m := range ok {
		if modelDeprecatesSamplingParams(m) {
			t.Errorf("expected %q to NOT be deprecated", m)
		}
	}
}
