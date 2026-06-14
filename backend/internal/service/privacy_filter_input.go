package service

import (
	"strconv"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// 隐私过滤复用内容审计的协议常量（ContentModerationProtocol*），无需新增。

type privacyEdit struct {
	path  string
	value string
}

// RewritePrivacyFilterBody 按协议遍历请求体中的文本节点并就地脱敏。
// redact 接收原始文本，返回（脱敏后文本, 是否改写）。仅改写文本字段，
// 不触碰 role / type / id / url 等结构字段。命中变化时用 sjson 保序写回。
func RewritePrivacyFilterBody(protocol string, body []byte, redact func(string) (string, bool)) []byte {
	if len(body) == 0 || redact == nil || !gjson.ValidBytes(body) {
		return body
	}
	var edits []privacyEdit
	switch protocol {
	case ContentModerationProtocolAnthropicMessages:
		collectMessagesEdits(body, redact, &edits)
		collectNodeEdits(gjson.GetBytes(body, "system"), "system", redact, &edits)
	case ContentModerationProtocolOpenAIChat:
		collectMessagesEdits(body, redact, &edits)
	case ContentModerationProtocolOpenAIResponses:
		collectResponsesInputEdits(body, redact, &edits)
		collectNodeEdits(gjson.GetBytes(body, "instructions"), "instructions", redact, &edits)
	case ContentModerationProtocolGemini:
		collectGeminiEdits(body, redact, &edits)
	case ContentModerationProtocolOpenAIImages:
		collectNodeEdits(gjson.GetBytes(body, "prompt"), "prompt", redact, &edits)
	default:
		collectMessagesEdits(body, redact, &edits)
		collectResponsesInputEdits(body, redact, &edits)
		collectGeminiEdits(body, redact, &edits)
	}
	if len(edits) == 0 {
		return body
	}
	out := body
	for _, e := range edits {
		if next, err := sjson.SetBytes(out, e.path, e.value); err == nil {
			out = next
		}
	}
	return out
}

// collectMessagesEdits 处理 OpenAI Chat / Anthropic 的 messages 数组（遍历全部消息）。
func collectMessagesEdits(body []byte, redact func(string) (string, bool), edits *[]privacyEdit) {
	messages := gjson.GetBytes(body, "messages")
	if !messages.IsArray() {
		return
	}
	for i, msg := range messages.Array() {
		path := "messages." + strconv.Itoa(i) + ".content"
		collectNodeEdits(msg.Get("content"), path, redact, edits)
	}
}

// collectResponsesInputEdits 处理 OpenAI Responses 的 input（字符串或数组）。
func collectResponsesInputEdits(body []byte, redact func(string) (string, bool), edits *[]privacyEdit) {
	input := gjson.GetBytes(body, "input")
	switch {
	case !input.Exists():
		return
	case input.Type == gjson.String:
		collectNodeEdits(input, "input", redact, edits)
	case input.IsArray():
		for i, item := range input.Array() {
			base := "input." + strconv.Itoa(i)
			if item.Get("content").Exists() {
				collectNodeEdits(item.Get("content"), base+".content", redact, edits)
			}
			if item.Get("text").Exists() && item.Get("text").Type == gjson.String {
				collectStringEdit(item.Get("text"), base+".text", redact, edits)
			}
		}
	}
}

// collectGeminiEdits 处理 Gemini 的 contents[].parts[].text 与 systemInstruction.parts[].text。
func collectGeminiEdits(body []byte, redact func(string) (string, bool), edits *[]privacyEdit) {
	contents := gjson.GetBytes(body, "contents")
	if contents.IsArray() {
		for i, content := range contents.Array() {
			parts := content.Get("parts")
			if !parts.IsArray() {
				continue
			}
			for j, part := range parts.Array() {
				if part.Get("text").Type == gjson.String {
					path := "contents." + strconv.Itoa(i) + ".parts." + strconv.Itoa(j) + ".text"
					collectStringEdit(part.Get("text"), path, redact, edits)
				}
			}
		}
	}
	for _, key := range []string{"systemInstruction", "system_instruction"} {
		parts := gjson.GetBytes(body, key+".parts")
		if !parts.IsArray() {
			continue
		}
		for j, part := range parts.Array() {
			if part.Get("text").Type == gjson.String {
				path := key + ".parts." + strconv.Itoa(j) + ".text"
				collectStringEdit(part.Get("text"), path, redact, edits)
			}
		}
	}
}

// collectNodeEdits 递归处理一个 content 节点：字符串 / 数组 / 对象。
func collectNodeEdits(node gjson.Result, path string, redact func(string) (string, bool), edits *[]privacyEdit) {
	switch {
	case !node.Exists():
		return
	case node.Type == gjson.String:
		collectStringEdit(node, path, redact, edits)
	case node.IsArray():
		for i, item := range node.Array() {
			itemPath := path + "." + strconv.Itoa(i)
			switch {
			case item.Type == gjson.String:
				collectStringEdit(item, itemPath, redact, edits)
			case item.IsObject():
				collectObjectEdits(item, itemPath, redact, edits)
			}
		}
	case node.IsObject():
		collectObjectEdits(node, path, redact, edits)
	}
}

// collectObjectEdits 处理 content part 对象：仅改写 text 字段并递归 content 字段。
func collectObjectEdits(obj gjson.Result, path string, redact func(string) (string, bool), edits *[]privacyEdit) {
	if text := obj.Get("text"); text.Type == gjson.String {
		collectStringEdit(text, path+".text", redact, edits)
	}
	if content := obj.Get("content"); content.Exists() {
		collectNodeEdits(content, path+".content", redact, edits)
	}
}

func collectStringEdit(node gjson.Result, path string, redact func(string) (string, bool), edits *[]privacyEdit) {
	original := node.String()
	if original == "" {
		return
	}
	if redacted, changed := redact(original); changed {
		*edits = append(*edits, privacyEdit{path: path, value: redacted})
	}
}
