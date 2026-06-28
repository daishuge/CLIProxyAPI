package handlers

import (
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const immersiveTranslateSegmentPromptTemplate = `当前请求来自沉浸式翻译插件。你正在返回机器会继续拆分、逐条贴回页面或字幕轨道的批量译文，不是普通整段翻译。用户体验优先级高于中文自然连贯性：必须先保证第 1 段原文对应第 1 段译文，第 2 段原文对应第 2 段译文，依次类推。

本次输入已经被独立分隔符行切成 %[2]d 个片段，分隔符行数量是 %[1]d。你的输出必须满足这个精确形状：
译文片段数量 = %[2]d
分隔符行数量 = %[1]d
只允许分隔符协议需要的空白行

工作方法：
1. 在心里把输入片段编号为 1 到 %[2]d，但不要输出编号。
2. 逐个编号翻译当前片段，只把当前片段的内容写入同编号译文位置。
3. 可以参考前后文理解代词、断句和语气，但禁止为了中文顺畅而改动片段边界、顺序或归属。
4. 如果一句话跨多个字幕片段，只翻译当前片段可见的文字；不要把上一段或下一段补进来。
5. 如果一个片段本身像标题、标签、HTML、用户名、帖子碎片、歌词碎片、日文助词、英文短语或只有标点，也必须保留一个独立译文位置。

分隔符协议：
1. 每个分隔符行必须只包含两个 ASCII 百分号：%%%%。
2. 分隔符必须严格复刻沉浸式翻译原始格式：前一个译文片段、一个空行、%%%%、一个空行、后一个译文片段，也就是“译文片段\n\n%%%%\n\n译文片段”。
3. 不要使用全角百分号，不要把 %%%% 包进引号、代码块、Markdown、项目符号或编号。
4. 分隔符行不能带句号、逗号、冒号、括号、引号、反引号、空格或解释文字；正确分隔符行只能是 %%%%。
5. 不要输出“译文片段\n%%%%\n译文片段”这种紧贴格式；沉浸式翻译可能无法拆分，用户会直接看到 %%%%。
6. 不要输出回车 U+000D，不要输出 CRLF，不要输出 \r，不要在 %%%% 前后加空格或 Tab。
7. 不要用空行、自然段换行、逗号、句号或任何其它符号代替 %%%% 分隔符；歌词、诗句和字幕短句也必须逐段输出 %%%%。

输出骨架必须是：
第 1 段译文

%%%%

第 2 段译文

%%%%

...
第 %[2]d 段译文

分段协议：
1. 一个输入片段只能产生一个译文片段。禁止合并相邻片段，禁止把一个片段拆成多个译文片段。
2. 可以参考上下文理解含义，但不得把相邻片段的意思搬进当前片段，不得重排顺序。
3. 片段即使是半句话、短语、标题、帖子碎片、歌词碎片、日文混合文本、已有中文或无需翻译内容，也必须保留它自己的输出位置。宁可中文不顺，也不要移动、合并或补写相邻片段。
4. 如果片段无需翻译，保留原文或只做必要本地化，但仍按原位置输出。
5. 每个译文片段内部尽量使用单个自然行；不要在片段内部插入空行，因为空行只允许服务于分隔符协议。
6. 不输出解释、标签、编号、项目符号、Markdown 加粗、额外标题或额外空段。

输出前只检查六件事：分隔符行必须等于 %[1]d；译文片段必须等于 %[2]d；第 N 段译文只对应第 N 段原文；每个分隔符行必须精确等于 %%%% ；每个分隔符必须是“\n\n%%%%\n\n”；整段输出不得包含 U+000D。`

func buildImmersiveTranslateSegmentPrompt(delimiterCount int) string {
	if delimiterCount < 1 {
		delimiterCount = 1
	}
	return fmt.Sprintf(immersiveTranslateSegmentPromptTemplate, delimiterCount, delimiterCount+1)
}

var immersiveTranslateSubtitlePrompt = buildImmersiveTranslateSegmentPrompt(1)

func immersiveTranslateSubtitlePromptForPayload(handlerType string, rawJSON []byte) (string, bool) {
	prompt, _, ok := immersiveTranslateSegmentPromptForPayload(handlerType, rawJSON)
	return prompt, ok
}

func immersiveTranslateSegmentPromptForPayload(handlerType string, rawJSON []byte) (string, int, bool) {
	if !strings.EqualFold(strings.TrimSpace(handlerType), "openai") || len(rawJSON) == 0 || !gjson.ValidBytes(rawJSON) {
		return "", 0, false
	}
	messages := gjson.GetBytes(rawJSON, "messages")
	if !messages.Exists() || !messages.IsArray() {
		return "", 0, false
	}

	var systemText strings.Builder
	var userText strings.Builder
	for _, message := range messages.Array() {
		role := strings.ToLower(strings.TrimSpace(message.Get("role").String()))
		content := openAIChatContentText(message.Get("content"))
		if strings.TrimSpace(content) == "" {
			continue
		}
		switch role {
		case "system":
			systemText.WriteString(content)
			systemText.WriteByte('\n')
		case "user":
			userText.WriteString(content)
			userText.WriteByte('\n')
		}
	}

	system := systemText.String()
	user := userText.String()
	delimiterCount := cleanPercentDelimiterLineCount(user)
	if !looksLikeImmersiveTranslateSegmentSystem(system) {
		return "", 0, false
	}
	if !looksLikeImmersiveTranslateSegmentUser(user, delimiterCount) {
		return "", 0, false
	}
	return buildImmersiveTranslateSegmentPrompt(delimiterCount), delimiterCount, true
}

func immersiveTranslateSegmentDelimiterCountForPayload(handlerType string, rawJSON []byte) (int, bool) {
	_, delimiterCount, ok := immersiveTranslateSegmentPromptForPayload(handlerType, rawJSON)
	return delimiterCount, ok
}

func normalizeImmersiveTranslateResponsePayload(handlerType string, payload []byte, delimiterCount int) []byte {
	if !strings.EqualFold(strings.TrimSpace(handlerType), "openai") || delimiterCount < 1 || len(payload) == 0 || !gjson.ValidBytes(payload) {
		return payload
	}
	choices := gjson.GetBytes(payload, "choices")
	if !choices.Exists() || !choices.IsArray() {
		return payload
	}

	out := payload
	changed := false
	for idx, choice := range choices.Array() {
		content := choice.Get("message.content")
		if !content.Exists() || content.Type != gjson.String {
			continue
		}
		normalized, ok := normalizeImmersiveTranslateContent(content.String(), delimiterCount)
		if !ok {
			continue
		}
		next, err := sjson.SetBytes(out, fmt.Sprintf("choices.%d.message.content", idx), normalized)
		if err != nil {
			continue
		}
		out = next
		changed = true
	}
	if !changed {
		return payload
	}
	return out
}

func normalizeImmersiveTranslateContent(content string, delimiterCount int) (string, bool) {
	if content == "" || delimiterCount < 1 {
		return content, false
	}
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	lines := strings.Split(normalized, "\n")
	if unwrapped, ok := unwrapWholeResponseMarkdownFence(lines); ok {
		lines = unwrapped
		normalized = strings.Join(lines, "\n")
	}

	count := 0
	changed := normalized != content
	for i, line := range lines {
		canonical, ok := canonicalImmersiveDelimiterLine(line)
		if !ok {
			continue
		}
		count++
		if line != canonical {
			lines[i] = canonical
			changed = true
		}
	}
	if count == 0 {
		if rebuilt, ok := rebuildMissingImmersiveDelimitersFromBlankGroups(lines, delimiterCount+1); ok {
			return rebuilt, true
		}
	}
	if count != delimiterCount || !changed {
		return content, false
	}
	return strings.Join(lines, "\n"), true
}

func canonicalImmersiveDelimiterLine(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return line, false
	}
	var percentCount int
	for _, r := range trimmed {
		switch r {
		case '%', '％':
			percentCount++
		case ' ', '\t', '\u3000':
			continue
		default:
			return line, false
		}
	}
	if percentCount < 2 {
		return line, false
	}
	return "%%", true
}

func rebuildMissingImmersiveDelimitersFromBlankGroups(lines []string, segmentCount int) (string, bool) {
	if segmentCount < 2 {
		return "", false
	}
	groups := make([]string, 0, segmentCount)
	current := make([]string, 0)
	flush := func() {
		if len(current) == 0 {
			return
		}
		groups = append(groups, strings.Join(current, "\n"))
		current = current[:0]
	}
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		current = append(current, line)
	}
	flush()
	if len(groups) != segmentCount {
		return "", false
	}
	return strings.Join(groups, "\n%%\n"), true
}

func unwrapWholeResponseMarkdownFence(lines []string) ([]string, bool) {
	start := -1
	end := -1
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			start = i
			break
		}
	}
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			end = i
			break
		}
	}
	if start < 0 || end <= start {
		return lines, false
	}
	if !strings.HasPrefix(strings.TrimSpace(lines[start]), "```") || strings.TrimSpace(lines[end]) != "```" {
		return lines, false
	}
	out := make([]string, 0, len(lines)-2)
	out = append(out, lines[:start]...)
	out = append(out, lines[start+1:end]...)
	out = append(out, lines[end+1:]...)
	return out, true
}

func looksLikeImmersiveTranslateSegmentSystem(text string) bool {
	if strings.TrimSpace(text) == "" {
		return false
	}
	if !strings.Contains(text, "你是一个专业的简体中文母语译者") {
		return false
	}
	if !strings.Contains(text, "返回的译文必须和原文保持完全相同的段落数量和格式") {
		return false
	}
	return strings.Contains(text, "Document Metadata:") || strings.Contains(text, "输入输出格式示例")
}

func looksLikeImmersiveTranslateSegmentUser(text string, delimiterCount int) bool {
	if strings.TrimSpace(text) == "" || delimiterCount == 0 {
		return false
	}
	return strings.Contains(text, "翻译为")
}

func cleanPercentDelimiterLineCount(text string) int {
	if text == "" {
		return 0
	}
	count := 0
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "%%" {
			count++
		}
	}
	return count
}

func openAIChatContentText(content gjson.Result) string {
	switch {
	case content.Type == gjson.String:
		return content.String()
	case content.IsArray():
		var builder strings.Builder
		for _, part := range content.Array() {
			if text := part.Get("text"); text.Exists() && text.Type == gjson.String {
				builder.WriteString(text.String())
				builder.WriteByte('\n')
			}
		}
		return builder.String()
	default:
		return ""
	}
}
