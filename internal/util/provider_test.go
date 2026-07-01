package util

import "testing"

func TestMaskSensitiveHeaderValueRedactsCookies(t *testing.T) {
	tests := []string{"Cookie", "cookie", "Set-Cookie", "X-Session-Cookie"}
	for _, key := range tests {
		t.Run(key, func(t *testing.T) {
			got := MaskSensitiveHeaderValue(key, "session=secret; other=value")
			if got != "<redacted>" {
				t.Fatalf("masked cookie = %q, want <redacted>", got)
			}
		})
	}
}

func TestMaskSensitiveHeaderValuePreservesAuthorizationScheme(t *testing.T) {
	got := MaskSensitiveHeaderValue("Authorization", "Bearer abcdefghijklmnop")
	if got != "Bearer abcd...mnop" {
		t.Fatalf("masked authorization = %q", got)
	}
}
