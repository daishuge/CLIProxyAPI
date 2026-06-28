package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	coreexecutor "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/executor"
	sdkconfig "github.com/router-for-me/CLIProxyAPI/v7/sdk/config"
	"github.com/tidwall/gjson"
)

const immersiveTranslateTestSystem = `你是一个专业的简体中文母语译者，需将文本流畅地翻译为简体中文。
## 翻译规则
1. 仅输出译文内容，禁止解释或添加任何额外内容
2. 返回的译文必须和原文保持完全相同的段落数量和格式

Document Metadata:
Title: 《Example - YouTube》
Type: Subtitle
Summary: This video explains a puzzle.`

func TestImmersiveTranslateSubtitlePromptInjectsOpenAIChat(t *testing.T) {
	executor := &presetPromptCaptureExecutor{provider: "immersive-translate-subtitle"}
	handler, model := newPresetPromptTestHandler(t, executor, sdkconfig.PresetPromptConfig{})
	body := immersiveTranslateTestBody(model, immersiveTranslateTestSystem, "翻译为简体中文：\n\nfirst line\n\n%%\n\nsecond line")

	_, _, errMsg := handler.ExecuteWithAuthManager(context.Background(), "openai", model, body, "")
	if errMsg != nil {
		t.Fatalf("ExecuteWithAuthManager returned error: %+v", errMsg)
	}

	payloads := executor.ExecutePayloads()
	if len(payloads) != 1 {
		t.Fatalf("captured upstream payloads = %d, want 1", len(payloads))
	}
	forwarded := payloads[0]
	if got := gjson.GetBytes(forwarded, "messages.0.role").String(); got != "system" {
		t.Fatalf("messages.0.role = %q, want system; payload=%s", got, forwarded)
	}
	if got := gjson.GetBytes(forwarded, "messages.0.content").String(); got != immersiveTranslateTestSystem {
		t.Fatalf("original system prompt moved incorrectly; got %q", got)
	}
	prompt := gjson.GetBytes(forwarded, "messages.1.content").String()
	if prompt != buildImmersiveTranslateSegmentPrompt(1) {
		t.Fatalf("subtitle prompt was not inserted after plugin system; got %q", prompt)
	}
	if !strings.Contains(prompt, "分隔符行数量是 1") || !strings.Contains(prompt, "译文片段数量 = 2") {
		t.Fatalf("subtitle prompt is missing exact shape counts: %q", prompt)
	}
	if !strings.Contains(prompt, "只允许分隔符协议需要的空白行") || !strings.Contains(prompt, "输出骨架必须是") {
		t.Fatalf("subtitle prompt is missing delimiter-owned blank-line skeleton constraints: %q", prompt)
	}
	if !strings.Contains(prompt, "ASCII 百分号：%%") || !strings.Contains(prompt, "精确等于 %%") {
		t.Fatalf("subtitle prompt lost literal %% delimiter text: %q", prompt)
	}
	if !strings.Contains(prompt, "不要输出 CRLF") || !strings.Contains(prompt, "U+000D") {
		t.Fatalf("subtitle prompt is missing line ending constraints: %q", prompt)
	}
	if !strings.Contains(prompt, "不要用空行") || !strings.Contains(prompt, "歌词") {
		t.Fatalf("subtitle prompt is missing blank-line replacement constraints: %q", prompt)
	}
	if !strings.Contains(prompt, "译文片段\\n\\n%%\\n\\n译文片段") || !strings.Contains(prompt, "每个分隔符必须是“\\n\\n%%\\n\\n”") {
		t.Fatalf("subtitle prompt is missing the plugin-compatible blank-wrapped delimiter shape: %q", prompt)
	}
	if !strings.Contains(prompt, "不要输出“译文片段\\n%%\\n译文片段”") || !strings.Contains(prompt, "用户会直接看到 %%") {
		t.Fatalf("subtitle prompt is missing visible-delimiter prevention constraints: %q", prompt)
	}
	if !strings.Contains(prompt, "把输入片段编号为 1 到 2") || !strings.Contains(prompt, "如果一句话跨多个字幕片段，只翻译当前片段可见的文字") {
		t.Fatalf("subtitle prompt is missing one-to-one subtitle alignment constraints: %q", prompt)
	}
	if !strings.Contains(prompt, "分隔符行不能带句号") || !strings.Contains(prompt, "第 N 段译文只对应第 N 段原文") {
		t.Fatalf("subtitle prompt is missing delimiter punctuation or final alignment checks: %q", prompt)
	}
	if !strings.Contains(prompt, "不要在片段内部插入空行") || !strings.Contains(prompt, "只允许服务于分隔符协议") {
		t.Fatalf("subtitle prompt is missing segment-internal blank-line checks: %q", prompt)
	}

	originals := executor.ExecuteOriginalRequests()
	if len(originals) != 1 || !bytes.Equal(originals[0], body) {
		t.Fatalf("OriginalRequest was mutated: got %q want %q", originals, body)
	}
}

func TestImmersiveTranslateSubtitlePromptDoesNotTriggerForInlinePercentOnly(t *testing.T) {
	executor := &presetPromptCaptureExecutor{provider: "immersive-translate-inline-percent"}
	handler, model := newPresetPromptTestHandler(t, executor, sdkconfig.PresetPromptConfig{})
	body := immersiveTranslateTestBody(model, immersiveTranslateTestSystem, "翻译为简体中文：\n\nopen http://a/%%30%30 now")

	_, _, errMsg := handler.ExecuteWithAuthManager(context.Background(), "openai", model, body, "")
	if errMsg != nil {
		t.Fatalf("ExecuteWithAuthManager returned error: %+v", errMsg)
	}
	payloads := executor.ExecutePayloads()
	if len(payloads) != 1 {
		t.Fatalf("captured upstream payloads = %d, want 1", len(payloads))
	}
	if got := gjson.GetBytes(payloads[0], "messages.1.content").String(); strings.Contains(got, "当前请求来自沉浸式翻译插件") {
		t.Fatalf("inline %% in URL triggered subtitle prompt unexpectedly; payload=%s", payloads[0])
	}
	if !bytes.Equal(payloads[0], body) {
		t.Fatalf("payload changed without a clean delimiter line: got %s want %s", payloads[0], body)
	}
}

func TestImmersiveTranslateSegmentPromptTriggersForNonYouTubeBatches(t *testing.T) {
	executor := &presetPromptCaptureExecutor{provider: "immersive-translate-generic"}
	handler, model := newPresetPromptTestHandler(t, executor, sdkconfig.PresetPromptConfig{})
	system := strings.ReplaceAll(immersiveTranslateTestSystem, "Title: 《Example - YouTube》", "Title: 《Reddit - The heart of the internet》")
	system = strings.ReplaceAll(system, "Type: Subtitle\n", "")
	body := immersiveTranslateTestBody(model, system, "翻译为简体中文：\n\nfirst line\n\n%%\n\nsecond line")

	_, _, errMsg := handler.ExecuteWithAuthManager(context.Background(), "openai", model, body, "")
	if errMsg != nil {
		t.Fatalf("ExecuteWithAuthManager returned error: %+v", errMsg)
	}
	payloads := executor.ExecutePayloads()
	if len(payloads) != 1 {
		t.Fatalf("captured upstream payloads = %d, want 1", len(payloads))
	}
	if got := gjson.GetBytes(payloads[0], "messages.1.content").String(); got != buildImmersiveTranslateSegmentPrompt(1) {
		t.Fatalf("non-YouTube segmented batch did not trigger segment prompt; got %q", got)
	}
}

func TestImmersiveTranslateSegmentPromptUsesExactDelimiterCounts(t *testing.T) {
	executor := &presetPromptCaptureExecutor{provider: "immersive-translate-counts"}
	handler, model := newPresetPromptTestHandler(t, executor, sdkconfig.PresetPromptConfig{})
	body := immersiveTranslateTestBody(model, immersiveTranslateTestSystem, "翻译为简体中文：\n\n1\n\n%%\n\n2\n\n%%\n\n3\n\n%%\n\n4\n\n%%\n\n5\n\n%%\n\n6\n\n%%\n\n7\n\n%%\n\n8\n\n%%\n\n9\n\n%%\n\n10")

	_, _, errMsg := handler.ExecuteWithAuthManager(context.Background(), "openai", model, body, "")
	if errMsg != nil {
		t.Fatalf("ExecuteWithAuthManager returned error: %+v", errMsg)
	}
	payloads := executor.ExecutePayloads()
	if len(payloads) != 1 {
		t.Fatalf("captured upstream payloads = %d, want 1", len(payloads))
	}
	prompt := gjson.GetBytes(payloads[0], "messages.1.content").String()
	if !strings.Contains(prompt, "分隔符行数量是 9") || !strings.Contains(prompt, "译文片段数量 = 10") {
		t.Fatalf("prompt did not include exact 9/10 shape counts: %q", prompt)
	}
	if !strings.Contains(prompt, "ASCII 百分号：%%") || strings.Contains(prompt, "ASCII 百分号：%。") {
		t.Fatalf("prompt does not preserve literal %% delimiter text: %q", prompt)
	}
}

func TestImmersiveTranslateSegmentPromptDoesNotTriggerForGenericChat(t *testing.T) {
	executor := &presetPromptCaptureExecutor{provider: "immersive-translate-generic-chat"}
	handler, model := newPresetPromptTestHandler(t, executor, sdkconfig.PresetPromptConfig{})
	body := []byte(fmt.Sprintf(`{"model":%q,"messages":[{"role":"system","content":"You are a translator."},{"role":"user","content":"翻译为简体中文：\n\nfirst line\n\n%%\n\nsecond line"}]}`, model))

	_, _, errMsg := handler.ExecuteWithAuthManager(context.Background(), "openai", model, body, "")
	if errMsg != nil {
		t.Fatalf("ExecuteWithAuthManager returned error: %+v", errMsg)
	}
	payloads := executor.ExecutePayloads()
	if len(payloads) != 1 {
		t.Fatalf("captured upstream payloads = %d, want 1", len(payloads))
	}
	if !bytes.Equal(payloads[0], body) {
		t.Fatalf("generic chat payload changed unexpectedly: got %s want %s", payloads[0], body)
	}
}

func TestImmersiveTranslateSubtitlePromptStacksWithPresetPromptAndRedactsLeaks(t *testing.T) {
	immersivePrompt := buildImmersiveTranslateSegmentPrompt(1)
	leaked := "prefix " + presetPromptTestMarker + " middle " + immersivePrompt + " suffix"
	executor := &presetPromptCaptureExecutor{
		provider:        "immersive-translate-redact",
		responsePayload: []byte(fmt.Sprintf(`{"choices":[{"message":{"content":%q}}]}`, leaked)),
	}
	handler, model := newPresetPromptTestHandler(t, executor, sdkconfig.PresetPromptConfig{
		Enabled: true,
		Prompt:  presetPromptTestMarker,
	})
	body := immersiveTranslateTestBody(model, immersiveTranslateTestSystem, "翻译为简体中文：\n\nfirst line\n\n%%\n\nsecond line")

	payload, _, errMsg := handler.ExecuteWithAuthManager(context.Background(), "openai", model, body, "")
	if errMsg != nil {
		t.Fatalf("ExecuteWithAuthManager returned error: %+v", errMsg)
	}
	if bytes.Contains(payload, []byte(presetPromptTestMarker)) || bytes.Contains(payload, []byte(immersiveTranslateSubtitlePrompt)) {
		t.Fatalf("response leaked injected prompt: %s", payload)
	}
	if got := bytes.Count(payload, []byte(presetPromptRedactionText)); got != 2 {
		t.Fatalf("redaction count = %d, want 2; payload=%s", got, payload)
	}

	payloads := executor.ExecutePayloads()
	if got := gjson.GetBytes(payloads[0], "messages.0.content").String(); got != presetPromptTestMarker {
		t.Fatalf("messages.0.content = %q, want preset prompt", got)
	}
	if got := gjson.GetBytes(payloads[0], "messages.1.content").String(); got != immersiveTranslateTestSystem {
		t.Fatalf("messages.1.content = %q, want original plugin prompt", got)
	}
	if got := gjson.GetBytes(payloads[0], "messages.2.content").String(); got != immersivePrompt {
		t.Fatalf("messages.2.content = %q, want subtitle prompt", got)
	}
}

func TestImmersiveTranslateResponseNormalizerConvertsCRLFDelimiters(t *testing.T) {
	response := "第一段\r\n%%\r\n第二段\r\n%%\r\n第三段"
	executor := &presetPromptCaptureExecutor{
		provider:        "immersive-translate-normalize-crlf",
		responsePayload: []byte(fmt.Sprintf(`{"choices":[{"message":{"content":%q}}]}`, response)),
	}
	handler, model := newPresetPromptTestHandler(t, executor, sdkconfig.PresetPromptConfig{})
	body := immersiveTranslateTestBody(model, immersiveTranslateTestSystem, "翻译为简体中文：\n\none\n\n%%\n\ntwo\n\n%%\n\nthree")

	payload, _, errMsg := handler.ExecuteWithAuthManager(context.Background(), "openai", model, body, "")
	if errMsg != nil {
		t.Fatalf("ExecuteWithAuthManager returned error: %+v", errMsg)
	}

	content := gjson.GetBytes(payload, "choices.0.message.content").String()
	if strings.Contains(content, "\r") {
		t.Fatalf("normalized response still contains carriage return: %q", content)
	}
	if got := cleanPercentDelimiterLineCount(content); got != 2 {
		t.Fatalf("delimiter count = %d, want 2; content=%q", got, content)
	}
	if content != "第一段\n%%\n第二段\n%%\n第三段" {
		t.Fatalf("content = %q, want exact LF-delimited response", content)
	}
}

func TestImmersiveTranslateResponseNormalizerTrimsDelimiterLineWhitespace(t *testing.T) {
	response := "第一段\n  %%  \n第二段"
	executor := &presetPromptCaptureExecutor{
		provider:        "immersive-translate-normalize-delimiter-spaces",
		responsePayload: []byte(fmt.Sprintf(`{"choices":[{"message":{"content":%q}}]}`, response)),
	}
	handler, model := newPresetPromptTestHandler(t, executor, sdkconfig.PresetPromptConfig{})
	body := immersiveTranslateTestBody(model, immersiveTranslateTestSystem, "翻译为简体中文：\n\none\n\n%%\n\ntwo")

	payload, _, errMsg := handler.ExecuteWithAuthManager(context.Background(), "openai", model, body, "")
	if errMsg != nil {
		t.Fatalf("ExecuteWithAuthManager returned error: %+v", errMsg)
	}

	content := gjson.GetBytes(payload, "choices.0.message.content").String()
	if content != "第一段\n%%\n第二段" {
		t.Fatalf("content = %q, want whitespace-free delimiter line", content)
	}
}

func TestImmersiveTranslateResponseNormalizerCanonicalizesPercentDelimiterVariants(t *testing.T) {
	response := "第一段\n％ ％\n第二段\n%%%%\n第三段"
	executor := &presetPromptCaptureExecutor{
		provider:        "immersive-translate-normalize-percent-variants",
		responsePayload: []byte(fmt.Sprintf(`{"choices":[{"message":{"content":%q}}]}`, response)),
	}
	handler, model := newPresetPromptTestHandler(t, executor, sdkconfig.PresetPromptConfig{})
	body := immersiveTranslateTestBody(model, immersiveTranslateTestSystem, "翻译为简体中文：\n\none\n\n%%\n\ntwo\n\n%%\n\nthree")

	payload, _, errMsg := handler.ExecuteWithAuthManager(context.Background(), "openai", model, body, "")
	if errMsg != nil {
		t.Fatalf("ExecuteWithAuthManager returned error: %+v", errMsg)
	}

	content := gjson.GetBytes(payload, "choices.0.message.content").String()
	if content != "第一段\n%%\n第二段\n%%\n第三段" {
		t.Fatalf("content = %q, want canonical percent delimiter lines", content)
	}
}

func TestImmersiveTranslateResponseNormalizerUnwrapsWholeResponseCodeFence(t *testing.T) {
	response := "```text\n第一段\n%%\n第二段\n```"
	executor := &presetPromptCaptureExecutor{
		provider:        "immersive-translate-normalize-code-fence",
		responsePayload: []byte(fmt.Sprintf(`{"choices":[{"message":{"content":%q}}]}`, response)),
	}
	handler, model := newPresetPromptTestHandler(t, executor, sdkconfig.PresetPromptConfig{})
	body := immersiveTranslateTestBody(model, immersiveTranslateTestSystem, "翻译为简体中文：\n\none\n\n%%\n\ntwo")

	payload, _, errMsg := handler.ExecuteWithAuthManager(context.Background(), "openai", model, body, "")
	if errMsg != nil {
		t.Fatalf("ExecuteWithAuthManager returned error: %+v", errMsg)
	}

	content := gjson.GetBytes(payload, "choices.0.message.content").String()
	if content != "第一段\n%%\n第二段" {
		t.Fatalf("content = %q, want unwrapped exact LF response", content)
	}
}

func TestImmersiveTranslateResponseNormalizerRebuildsMissingDelimitersFromBlankGroups(t *testing.T) {
	response := "我数着海湾上方的灯光\n\n却忘掉了我想说的话\n\n副歌又回来了又一次\n\n但却没有答案。"
	executor := &presetPromptCaptureExecutor{
		provider:        "immersive-translate-normalize-missing-delimiters",
		responsePayload: []byte(fmt.Sprintf(`{"choices":[{"message":{"content":%q}}]}`, response)),
	}
	handler, model := newPresetPromptTestHandler(t, executor, sdkconfig.PresetPromptConfig{})
	body := immersiveTranslateTestBody(model, immersiveTranslateTestSystem, "翻译为简体中文：\n\nI count the lights above the bay\n\n%%\n\nThen forget what I meant to say\n\n%%\n\nThe chorus comes back once again\n\n%%\n\nBut still no answer at the end")

	payload, _, errMsg := handler.ExecuteWithAuthManager(context.Background(), "openai", model, body, "")
	if errMsg != nil {
		t.Fatalf("ExecuteWithAuthManager returned error: %+v", errMsg)
	}

	content := gjson.GetBytes(payload, "choices.0.message.content").String()
	if content != "我数着海湾上方的灯光\n%%\n却忘掉了我想说的话\n%%\n副歌又回来了又一次\n%%\n但却没有答案。" {
		t.Fatalf("content = %q, want rebuilt delimiter lines between blank-line groups", content)
	}
}

func TestImmersiveTranslateResponseNormalizerDoesNotRebuildAmbiguousBlankGroups(t *testing.T) {
	response := "第一段\n\n第二段\n\n第三段"
	executor := &presetPromptCaptureExecutor{
		provider:        "immersive-translate-normalize-ambiguous-blank-groups",
		responsePayload: []byte(fmt.Sprintf(`{"choices":[{"message":{"content":%q}}]}`, response)),
	}
	handler, model := newPresetPromptTestHandler(t, executor, sdkconfig.PresetPromptConfig{})
	body := immersiveTranslateTestBody(model, immersiveTranslateTestSystem, "翻译为简体中文：\n\none\n\n%%\n\ntwo")

	payload, _, errMsg := handler.ExecuteWithAuthManager(context.Background(), "openai", model, body, "")
	if errMsg != nil {
		t.Fatalf("ExecuteWithAuthManager returned error: %+v", errMsg)
	}

	content := gjson.GetBytes(payload, "choices.0.message.content").String()
	if content != response {
		t.Fatalf("content = %q, want ambiguous blank groups left untouched", content)
	}
}

func TestImmersiveTranslateResponseNormalizerLeavesGenericChatUntouched(t *testing.T) {
	response := "第一段\r\n%%\r\n第二段"
	executor := &presetPromptCaptureExecutor{
		provider:        "immersive-translate-normalize-generic-chat",
		responsePayload: []byte(fmt.Sprintf(`{"choices":[{"message":{"content":%q}}]}`, response)),
	}
	handler, model := newPresetPromptTestHandler(t, executor, sdkconfig.PresetPromptConfig{})
	body := []byte(fmt.Sprintf(`{"model":%q,"messages":[{"role":"system","content":"You are a translator."},{"role":"user","content":"翻译为简体中文：\n\none\n\n%%\n\ntwo"}]}`, model))

	payload, _, errMsg := handler.ExecuteWithAuthManager(context.Background(), "openai", model, body, "")
	if errMsg != nil {
		t.Fatalf("ExecuteWithAuthManager returned error: %+v", errMsg)
	}

	content := gjson.GetBytes(payload, "choices.0.message.content").String()
	if content != response {
		t.Fatalf("generic response changed: got %q want %q", content, response)
	}
}

func TestImmersiveTranslateSubtitlePromptStreamingRedactsLeakAcrossChunks(t *testing.T) {
	immersivePrompt := buildImmersiveTranslateSegmentPrompt(1)
	leaked := "prefix " + immersivePrompt + " suffix"
	encodedLeak, err := json.Marshal(leaked)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	frame := []byte("data: {\"delta\":" + string(encodedLeak) + "}\n\n")
	mid := len(frame) / 2
	executor := &presetPromptCaptureExecutor{
		provider: "immersive-translate-stream-redact",
		streamChunks: []coreexecutor.StreamChunk{
			{Payload: bytes.Clone(frame[:mid])},
			{Payload: bytes.Clone(frame[mid:])},
		},
	}
	handler, model := newPresetPromptTestHandler(t, executor, sdkconfig.PresetPromptConfig{})
	body := immersiveTranslateTestBody(model, immersiveTranslateTestSystem, "翻译为简体中文：\n\nfirst line\n\n%%\n\nsecond line")

	data, _, errs := handler.ExecuteStreamWithAuthManager(context.Background(), "openai", model, body, "")
	var got bytes.Buffer
	for chunk := range data {
		got.Write(chunk)
	}
	for errMsg := range errs {
		if errMsg != nil {
			t.Fatalf("unexpected stream error: %+v", errMsg)
		}
	}
	if strings.Contains(got.String(), "当前请求来自沉浸式翻译") {
		t.Fatalf("stream response leaked subtitle prompt: %s", got.String())
	}
	if !strings.Contains(got.String(), presetPromptRedactionText) {
		t.Fatalf("stream response missing redaction marker: %s", got.String())
	}
}

func immersiveTranslateTestBody(model, system, user string) []byte {
	return []byte(fmt.Sprintf(`{"model":%q,"messages":[{"role":"system","content":%q},{"role":"user","content":%q}]}`, model, system, user))
}
