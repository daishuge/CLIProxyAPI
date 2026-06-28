package handlers

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/config"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"golang.org/x/net/context"
)

const presetPromptSeparator = "\n\n"
const presetPromptRedactionText = "[preset prompt redacted]"

// SetPresetPromptConfig updates the request-time preset prompt snapshot.
func (h *BaseAPIHandler) SetPresetPromptConfig(cfg config.PresetPromptConfig) {
	if h == nil {
		return
	}
	cfg.Normalize()
	h.presetPromptMu.Lock()
	h.presetPromptConfig = cfg
	h.presetPromptMu.Unlock()
}

// SetAPIKeyControls updates per-client-key request-time controls used by handlers.
func (h *BaseAPIHandler) SetAPIKeyControls(controls []config.APIKeyControl) {
	if h == nil {
		return
	}
	cloned := cloneAPIKeyControls(controls)
	h.presetPromptMu.Lock()
	h.apiKeyControls = cloned
	h.presetPromptMu.Unlock()
}

func (h *BaseAPIHandler) activePresetPrompt() (string, bool) {
	return h.activePresetPromptForAPIKey("")
}

func (h *BaseAPIHandler) activePresetPromptForAPIKey(apiKey string) (string, bool) {
	if h == nil {
		return "", false
	}
	h.presetPromptMu.RLock()
	cfg := h.presetPromptConfig
	if control := findPresetPromptAPIKeyControl(h.apiKeyControls, apiKey); control != nil && control.PresetPrompt != nil {
		cfg = *control.PresetPrompt
	}
	h.presetPromptMu.RUnlock()
	return activePresetPromptFromConfig(cfg)
}

func activePresetPromptFromConfig(cfg config.PresetPromptConfig) (string, bool) {
	if !cfg.Enabled || strings.TrimSpace(cfg.Prompt) == "" {
		return "", false
	}
	maxBytes := cfg.MaxBytes
	if maxBytes <= 0 {
		maxBytes = config.DefaultPresetPromptMaxBytes
	}
	if maxBytes > config.PresetPromptHardMaxBytes {
		maxBytes = config.PresetPromptHardMaxBytes
	}
	if len([]byte(cfg.Prompt)) > maxBytes {
		return "", false
	}
	return cfg.Prompt, true
}

func (h *BaseAPIHandler) applyPresetPromptToPayload(handlerType string, rawJSON []byte) []byte {
	return h.applyPresetPromptToPayloadForAPIKey(handlerType, rawJSON, "")
}

func (h *BaseAPIHandler) applyPresetPromptToPayloadForAPIKey(handlerType string, rawJSON []byte, apiKey string) []byte {
	prompt, ok := h.activePresetPromptForAPIKey(apiKey)
	if !ok || len(rawJSON) == 0 || !json.Valid(rawJSON) {
		return rawJSON
	}

	switch strings.ToLower(strings.TrimSpace(handlerType)) {
	case "openai":
		return injectPresetPromptIntoOpenAIChat(rawJSON, prompt)
	case "openai-response":
		return injectPresetPromptIntoOpenAIResponses(rawJSON, prompt)
	case "claude":
		return injectPresetPromptIntoClaude(rawJSON, prompt)
	case "gemini", "gemini-cli":
		return injectPresetPromptIntoGemini(rawJSON, prompt)
	default:
		return rawJSON
	}
}

// applyRequestPromptInjectionsToPayloadForAPIKey performs the chained request-time
// prompt injections (preset prompt followed by immersive-translate alignment prompt).
// The two injections are orthogonal: the preset prompt is prepended as a leading
// system message, while the immersive-translate prompt is inserted after any leading
// system/developer messages. It returns the mutated payload together with the list of
// injected prompts that must be redacted from upstream responses.
func (h *BaseAPIHandler) applyRequestPromptInjectionsToPayloadForAPIKey(handlerType string, rawJSON []byte, apiKey string) ([]byte, []string) {
	payload := rawJSON
	var redactions []string

	if prompt, ok := h.activePresetPromptForAPIKey(apiKey); ok {
		mutated := applyPromptToPayload(handlerType, payload, prompt)
		if !bytes.Equal(mutated, payload) {
			payload = mutated
			redactions = append(redactions, prompt)
		}
	}

	if prompt, ok := immersiveTranslateSubtitlePromptForPayload(handlerType, rawJSON); ok {
		mutated := applyImmersiveTranslatePromptToPayload(handlerType, payload, prompt)
		if !bytes.Equal(mutated, payload) {
			payload = mutated
			redactions = append(redactions, prompt)
		}
	}

	return payload, redactions
}

func applyImmersiveTranslatePromptToPayload(handlerType string, rawJSON []byte, prompt string) []byte {
	if strings.EqualFold(strings.TrimSpace(handlerType), "openai") {
		return injectPromptAfterLeadingSystemOpenAIChat(rawJSON, prompt)
	}
	return applyPromptToPayload(handlerType, rawJSON, prompt)
}

func applyPromptToPayload(handlerType string, rawJSON []byte, prompt string) []byte {
	if strings.TrimSpace(prompt) == "" || len(rawJSON) == 0 || !json.Valid(rawJSON) {
		return rawJSON
	}
	switch strings.ToLower(strings.TrimSpace(handlerType)) {
	case "openai":
		return injectPresetPromptIntoOpenAIChat(rawJSON, prompt)
	case "openai-response":
		return injectPresetPromptIntoOpenAIResponses(rawJSON, prompt)
	case "claude":
		return injectPresetPromptIntoClaude(rawJSON, prompt)
	case "gemini", "gemini-cli":
		return injectPresetPromptIntoGemini(rawJSON, prompt)
	default:
		return rawJSON
	}
}

func redactPresetPromptFromPayload(payload []byte, prompt string) []byte {
	needles := presetPromptNeedles(prompt)
	if len(payload) == 0 || len(needles) == 0 {
		return payload
	}
	out := bytes.Clone(payload)
	replacement := []byte(presetPromptRedactionText)
	for _, needle := range needles {
		out = bytes.ReplaceAll(out, needle, replacement)
	}
	return out
}

func redactPresetPromptsFromPayload(payload []byte, prompts []string) []byte {
	if len(payload) == 0 || len(prompts) == 0 {
		return payload
	}
	out := payload
	for _, prompt := range prompts {
		out = redactPresetPromptFromPayload(out, prompt)
	}
	return out
}

type presetPromptStreamRedactor struct {
	needles [][]byte
	buffer  []byte
}

func newPresetPromptStreamRedactor(prompt string) *presetPromptStreamRedactor {
	return newPresetPromptStreamRedactorForPrompts([]string{prompt})
}

func newPresetPromptStreamRedactorForPrompts(prompts []string) *presetPromptStreamRedactor {
	seen := map[string]struct{}{}
	var needles [][]byte
	for _, prompt := range prompts {
		for _, needle := range presetPromptNeedles(prompt) {
			key := string(needle)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			needles = append(needles, needle)
		}
	}
	if len(needles) == 0 {
		return nil
	}
	return &presetPromptStreamRedactor{needles: needles}
}

// Push appends the incoming chunk to the internal buffer, redacts any preset prompt
// needles present, and returns only the bytes that form complete SSE frames. Bytes
// belonging to a partially received frame are retained so that a needle straddling a
// frame boundary (for example a base64 or JSON-escaped run split across chunks) is not
// emitted before it can be redacted.
func (r *presetPromptStreamRedactor) Push(chunk []byte) []byte {
	if r == nil || len(chunk) == 0 {
		return chunk
	}
	r.buffer = append(r.buffer, chunk...)
	r.replaceBuffered()
	emitLen := completeSSEPrefixLen(r.buffer)
	if emitLen == 0 {
		return nil
	}
	out := bytes.Clone(r.buffer[:emitLen])
	r.buffer = append(r.buffer[:0], r.buffer[emitLen:]...)
	return out
}

// Flush redacts and returns any bytes still buffered when the upstream stream ends.
func (r *presetPromptStreamRedactor) Flush() []byte {
	if r == nil || len(r.buffer) == 0 {
		return nil
	}
	r.replaceBuffered()
	out := bytes.Clone(r.buffer)
	r.buffer = nil
	return out
}

func (r *presetPromptStreamRedactor) replaceBuffered() {
	if r == nil || len(r.buffer) == 0 {
		return
	}
	replacement := []byte(presetPromptRedactionText)
	for _, needle := range r.needles {
		r.buffer = bytes.ReplaceAll(r.buffer, needle, replacement)
	}
}

// completeSSEPrefixLen returns the length of the buffer prefix that ends on the last
// SSE frame boundary ("\n\n"). This boundary-aware accounting lets the redactor emit
// only whole frames and keep the trailing partial frame buffered until it completes.
func completeSSEPrefixLen(buf []byte) int {
	last := 0
	searchStart := 0
	for searchStart <= len(buf) {
		idx := bytes.Index(buf[searchStart:], []byte("\n\n"))
		if idx < 0 {
			break
		}
		last = searchStart + idx + 2
		searchStart = last
	}
	return last
}

func presetPromptNeedles(prompt string) [][]byte {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return nil
	}
	seen := map[string]struct{}{}
	add := func(value []byte, out *[][]byte) {
		if len(value) == 0 {
			return
		}
		key := string(value)
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		*out = append(*out, bytes.Clone(value))
	}
	out := make([][]byte, 0, 2)
	add([]byte(prompt), &out)
	if encoded, err := json.Marshal(prompt); err == nil && len(encoded) >= 2 {
		add(encoded[1:len(encoded)-1], &out)
	}
	return out
}

// apiKeyFromRequestContext extracts the inbound client API key from the embedded gin
// context. In v7 the access middleware stores the authenticated principal under the
// "userApiKey" gin key (see internal/api/server.go AuthMiddleware), so per-key preset
// prompt overrides are resolved against that value.
func apiKeyFromRequestContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	ginCtx, ok := ctx.Value("gin").(*gin.Context)
	if !ok || ginCtx == nil {
		return ""
	}
	return strings.TrimSpace(ginCtx.GetString("userApiKey"))
}

func cloneAPIKeyControls(controls []config.APIKeyControl) []config.APIKeyControl {
	if len(controls) == 0 {
		return nil
	}
	out := make([]config.APIKeyControl, len(controls))
	for i := range controls {
		out[i] = controls[i]
		if controls[i].PresetPrompt != nil {
			cfg := *controls[i].PresetPrompt
			cfg.Normalize()
			out[i].PresetPrompt = &cfg
		}
	}
	return out
}

func findPresetPromptAPIKeyControl(controls []config.APIKeyControl, apiKey string) *config.APIKeyControl {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil
	}
	for i := range controls {
		key := strings.TrimSpace(controls[i].APIKey)
		if key == "" {
			key = strings.TrimSpace(controls[i].Key)
		}
		if key == apiKey {
			return &controls[i]
		}
	}
	return nil
}

func injectPresetPromptIntoOpenAIChat(rawJSON []byte, prompt string) []byte {
	messages := gjson.GetBytes(rawJSON, "messages")
	if !messages.Exists() || !messages.IsArray() {
		return rawJSON
	}
	item, err := json.Marshal(map[string]any{
		"role":    "system",
		"content": prompt,
	})
	if err != nil {
		return rawJSON
	}
	mutatedMessages, ok := prependJSONRawArray(messages.Raw, item)
	if !ok {
		return rawJSON
	}
	out, err := sjson.SetRawBytes(rawJSON, "messages", mutatedMessages)
	if err != nil {
		return rawJSON
	}
	return out
}

func injectPromptAfterLeadingSystemOpenAIChat(rawJSON []byte, prompt string) []byte {
	messages := gjson.GetBytes(rawJSON, "messages")
	if !messages.Exists() || !messages.IsArray() {
		return rawJSON
	}
	item, err := json.Marshal(map[string]any{
		"role":    "system",
		"content": prompt,
	})
	if err != nil {
		return rawJSON
	}

	var rawMessages []json.RawMessage
	if err := json.Unmarshal([]byte(messages.Raw), &rawMessages); err != nil {
		return rawJSON
	}
	insertAt := 0
	for insertAt < len(rawMessages) {
		role := strings.ToLower(strings.TrimSpace(gjson.GetBytes(rawMessages[insertAt], "role").String()))
		if role != "system" && role != "developer" {
			break
		}
		insertAt++
	}

	mutated := make([]json.RawMessage, 0, len(rawMessages)+1)
	mutated = append(mutated, rawMessages[:insertAt]...)
	mutated = append(mutated, json.RawMessage(item))
	mutated = append(mutated, rawMessages[insertAt:]...)
	mutatedMessages, err := json.Marshal(mutated)
	if err != nil {
		return rawJSON
	}
	out, err := sjson.SetRawBytes(rawJSON, "messages", mutatedMessages)
	if err != nil {
		return rawJSON
	}
	return out
}

func injectPresetPromptIntoOpenAIResponses(rawJSON []byte, prompt string) []byte {
	if openAIResponsesHasImageGenerationTool(rawJSON) {
		return rawJSON
	}
	instructions := gjson.GetBytes(rawJSON, "instructions")
	input := gjson.GetBytes(rawJSON, "input")
	if !instructions.Exists() && !input.Exists() {
		return rawJSON
	}
	if instructions.Exists() {
		if instructions.Type != gjson.String {
			return rawJSON
		}
		out, err := sjson.SetBytes(rawJSON, "instructions", prompt+presetPromptSeparator+instructions.String())
		if err != nil {
			return rawJSON
		}
		return out
	}
	out, err := sjson.SetBytes(rawJSON, "instructions", prompt)
	if err != nil {
		return rawJSON
	}
	return out
}

func openAIResponsesHasImageGenerationTool(rawJSON []byte) bool {
	tools := gjson.GetBytes(rawJSON, "tools")
	if !tools.Exists() || !tools.IsArray() {
		return false
	}
	for _, tool := range tools.Array() {
		if strings.EqualFold(strings.TrimSpace(tool.Get("type").String()), "image_generation") {
			return true
		}
	}
	return false
}

func injectPresetPromptIntoClaude(rawJSON []byte, prompt string) []byte {
	messages := gjson.GetBytes(rawJSON, "messages")
	if !messages.Exists() || !messages.IsArray() {
		return rawJSON
	}

	system := gjson.GetBytes(rawJSON, "system")
	if !system.Exists() {
		out, err := sjson.SetBytes(rawJSON, "system", prompt)
		if err != nil {
			return rawJSON
		}
		return out
	}
	switch {
	case system.Type == gjson.String:
		out, err := sjson.SetBytes(rawJSON, "system", prompt+presetPromptSeparator+system.String())
		if err != nil {
			return rawJSON
		}
		return out
	case system.IsArray():
		item, err := json.Marshal(map[string]any{
			"type": "text",
			"text": prompt,
		})
		if err != nil {
			return rawJSON
		}
		mutatedSystem, ok := prependJSONRawArray(system.Raw, item)
		if !ok {
			return rawJSON
		}
		out, err := sjson.SetRawBytes(rawJSON, "system", mutatedSystem)
		if err != nil {
			return rawJSON
		}
		return out
	default:
		return rawJSON
	}
}

func injectPresetPromptIntoGemini(rawJSON []byte, prompt string) []byte {
	contents := gjson.GetBytes(rawJSON, "contents")
	if !contents.Exists() || !contents.IsArray() {
		return rawJSON
	}

	part, err := json.Marshal(map[string]any{"text": prompt})
	if err != nil {
		return rawJSON
	}
	systemInstruction := gjson.GetBytes(rawJSON, "systemInstruction")
	if !systemInstruction.Exists() {
		item, err := json.Marshal(map[string]any{
			"parts": []map[string]any{{"text": prompt}},
		})
		if err != nil {
			return rawJSON
		}
		out, err := sjson.SetRawBytes(rawJSON, "systemInstruction", item)
		if err != nil {
			return rawJSON
		}
		return out
	}
	if !systemInstruction.IsObject() {
		return rawJSON
	}

	parts := gjson.GetBytes(rawJSON, "systemInstruction.parts")
	if !parts.Exists() {
		out, err := sjson.SetRawBytes(rawJSON, "systemInstruction.parts", []byte("["+string(part)+"]"))
		if err != nil {
			return rawJSON
		}
		return out
	}
	if !parts.IsArray() {
		return rawJSON
	}
	mutatedParts, ok := prependJSONRawArray(parts.Raw, part)
	if !ok {
		return rawJSON
	}
	out, err := sjson.SetRawBytes(rawJSON, "systemInstruction.parts", mutatedParts)
	if err != nil {
		return rawJSON
	}
	return out
}

func prependJSONRawArray(arrayRaw string, item []byte) ([]byte, bool) {
	trimmed := strings.TrimSpace(arrayRaw)
	if trimmed == "" || !strings.HasPrefix(trimmed, "[") || !strings.HasSuffix(trimmed, "]") {
		return nil, false
	}
	if strings.TrimSpace(trimmed[1:len(trimmed)-1]) == "" {
		out := make([]byte, 0, len(item)+2)
		out = append(out, '[')
		out = append(out, item...)
		out = append(out, ']')
		return out, true
	}
	out := make([]byte, 0, len(trimmed)+len(item)+1)
	out = append(out, '[')
	out = append(out, item...)
	out = append(out, ',')
	out = append(out, trimmed[1:]...)
	return out, true
}
