package service

import (
	"strings"
	"testing"

	"github.com/tidwall/gjson"
)

func allBuiltinsEnabled() map[string]bool {
	m := map[string]bool{}
	for _, id := range PrivacyFilterBuiltinIDs() {
		m[id] = true
	}
	return m
}

func TestApplyPrivacyRules_Builtins(t *testing.T) {
	rules := compilePrivacyRules(allBuiltinsEnabled(), nil)
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"email", "reach me at john.doe@example.com please", "reach me at [EMAIL] please"},
		{"phone", "call 13800138000 now", "call [PHONE] now"},
		{"id_card", "id 11010519491231002X end", "id [ID_CARD] end"},
		{"ipv4", "server 192.168.1.1 down", "server [IP] down"},
		{"secret", "key sk-abcdefghijklmnop1234 leak", "key [SECRET] leak"},
		{"none", "nothing sensitive here", "nothing sensitive here"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, changed := applyPrivacyRules(rules, tc.in)
			if got != tc.want {
				t.Fatalf("redact(%q) = %q, want %q", tc.in, got, tc.want)
			}
			if changed == (tc.in == tc.want) {
				t.Fatalf("changed flag wrong for %q: got %v", tc.in, changed)
			}
		})
	}
}

func TestApplyPrivacyRules_BuiltinToggle(t *testing.T) {
	enabled := allBuiltinsEnabled()
	enabled[PrivacyFilterBuiltinEmail] = false
	rules := compilePrivacyRules(enabled, nil)
	got, _ := applyPrivacyRules(rules, "mail a@b.com phone 13800138000")
	if !strings.Contains(got, "a@b.com") {
		t.Fatalf("email rule disabled but redacted: %q", got)
	}
	if strings.Contains(got, "13800138000") {
		t.Fatalf("phone should still be redacted: %q", got)
	}
}

func TestApplyPrivacyRules_CustomRule(t *testing.T) {
	custom := []PrivacyFilterRule{{Name: "ticket", Pattern: `TICKET-\d+`, Replacement: "[TICKET]", Enabled: true}}
	rules := compilePrivacyRules(map[string]bool{}, custom)
	got, changed := applyPrivacyRules(rules, "see TICKET-12345")
	if got != "see [TICKET]" || !changed {
		t.Fatalf("custom rule failed: %q changed=%v", got, changed)
	}
}

func redactAll(text string) (string, bool) {
	rules := compilePrivacyRules(allBuiltinsEnabled(), nil)
	return applyPrivacyRules(rules, text)
}

func TestRewritePrivacyFilterBody_OpenAIChat(t *testing.T) {
	body := []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":"email me@example.com"},{"role":"assistant","content":"ok"}]}`)
	out := RewritePrivacyFilterBody(ContentModerationProtocolOpenAIChat, body, redactAll)
	if gjson.GetBytes(out, "model").String() != "gpt-4o" {
		t.Fatalf("model field corrupted: %s", out)
	}
	if gjson.GetBytes(out, "messages.0.role").String() != "user" {
		t.Fatalf("role field corrupted: %s", out)
	}
	if got := gjson.GetBytes(out, "messages.0.content").String(); got != "email [EMAIL]" {
		t.Fatalf("content not redacted: %q", got)
	}
}

func TestRewritePrivacyFilterBody_OpenAIChatContentParts(t *testing.T) {
	body := []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":[{"type":"text","text":"call 13800138000"},{"type":"image_url","image_url":{"url":"https://x/y.png"}}]}]}`)
	out := RewritePrivacyFilterBody(ContentModerationProtocolOpenAIChat, body, redactAll)
	if got := gjson.GetBytes(out, "messages.0.content.0.text").String(); got != "call [PHONE]" {
		t.Fatalf("text part not redacted: %q", got)
	}
	if got := gjson.GetBytes(out, "messages.0.content.1.image_url.url").String(); got != "https://x/y.png" {
		t.Fatalf("image url corrupted: %q", got)
	}
}

func TestRewritePrivacyFilterBody_Gemini(t *testing.T) {
	body := []byte(`{"contents":[{"role":"user","parts":[{"text":"id 11010519491231002X"}]}]}`)
	out := RewritePrivacyFilterBody(ContentModerationProtocolGemini, body, redactAll)
	if got := gjson.GetBytes(out, "contents.0.parts.0.text").String(); got != "id [ID_CARD]" {
		t.Fatalf("gemini text not redacted: %q", got)
	}
}

func TestRewritePrivacyFilterBody_NoChangeReturnsOriginal(t *testing.T) {
	body := []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":"hello world"}]}`)
	out := RewritePrivacyFilterBody(ContentModerationProtocolOpenAIChat, body, redactAll)
	if string(out) != string(body) {
		t.Fatalf("expected unchanged body, got %s", out)
	}
}

func TestPrivacyFilterConfig_Normalize(t *testing.T) {
	cfg := defaultPrivacyFilterConfig()
	if len(cfg.BuiltinRules) != len(PrivacyFilterBuiltinIDs()) {
		t.Fatalf("default builtins incomplete: %v", cfg.BuiltinRules)
	}
	cfg.BuiltinRules["bogus_unknown"] = true
	cfg.CustomRules = []PrivacyFilterRule{{Pattern: "  ", Enabled: true}, {Pattern: "x+", Enabled: true}}
	cfg.normalize()
	if _, ok := cfg.BuiltinRules["bogus_unknown"]; ok {
		t.Fatalf("unknown builtin id not dropped")
	}
	if len(cfg.CustomRules) != 1 {
		t.Fatalf("blank custom rule not dropped: %v", cfg.CustomRules)
	}
}
