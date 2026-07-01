package common

import "encoding/json"

var unsupportedGeminiSchemaKeys = map[string]struct{}{
	"$comment":         {},
	"enumDescriptions": {},
}

// SanitizeJSONSchemaRaw removes JSON Schema fields known to be rejected by Gemini.
// The input is returned unchanged if it cannot be parsed as JSON.
func SanitizeJSONSchemaRaw(raw string) string {
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return raw
	}
	sanitizeJSONSchemaValue(value)
	out, err := json.Marshal(value)
	if err != nil {
		return raw
	}
	return string(out)
}

func sanitizeJSONSchemaValue(value any) {
	switch typed := value.(type) {
	case map[string]any:
		for key := range unsupportedGeminiSchemaKeys {
			delete(typed, key)
		}
		for _, child := range typed {
			sanitizeJSONSchemaValue(child)
		}
	case []any:
		for _, child := range typed {
			sanitizeJSONSchemaValue(child)
		}
	}
}
