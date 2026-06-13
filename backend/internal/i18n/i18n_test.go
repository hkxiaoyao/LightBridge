package i18n

import "testing"

func TestParseLang(t *testing.T) {
	cases := map[string]Lang{
		"":        LangEN,
		"en":      LangEN,
		"en-US":   LangEN,
		"zh":      LangZH,
		"zh-CN":   LangZH,
		"zh-Hans": LangZH,
		"  ZH  ":  LangZH,
		"fr":      LangEN, // unsupported falls back to English
		"de-DE":   LangEN,
	}
	for in, want := range cases {
		if got := ParseLang(in); got != want {
			t.Errorf("ParseLang(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestResolveAcceptLanguage(t *testing.T) {
	// Default is English unless overridden.
	SetDefault(LangEN)
	t.Cleanup(func() { SetDefault(LangEN) })

	cases := map[string]Lang{
		"":                        LangEN, // empty -> default
		"zh-CN,zh;q=0.9,en;q=0.8": LangZH,
		"en-US,en;q=0.9":          LangEN,
		"fr-FR,fr;q=0.9,zh;q=0.5": LangZH, // first supported tag wins
		"fr-FR,de;q=0.9":          LangEN, // no supported tag -> default
		"zh":                      LangZH,
		" en ":                    LangEN,
	}
	for header, want := range cases {
		if got := ResolveAcceptLanguage(header); got != want {
			t.Errorf("ResolveAcceptLanguage(%q) = %q, want %q", header, got, want)
		}
	}

	// When no supported tag is present, the configured default applies.
	SetDefault(LangZH)
	if got := ResolveAcceptLanguage("fr-FR,de;q=0.9"); got != LangZH {
		t.Errorf("with zh default, ResolveAcceptLanguage(unsupported) = %q, want zh", got)
	}
	if got := ResolveAcceptLanguage(""); got != LangZH {
		t.Errorf("with zh default, ResolveAcceptLanguage(empty) = %q, want zh", got)
	}
}

func TestTranslate(t *testing.T) {
	// English returns the key verbatim.
	if got := Translate(LangEN, "No available accounts"); got != "No available accounts" {
		t.Errorf("Translate(en) = %q", got)
	}
	// Chinese returns the catalog value.
	if got := Translate(LangZH, "No available accounts"); got != "无可用账号" {
		t.Errorf("Translate(zh) = %q, want 无可用账号", got)
	}
	// Unknown messages pass through unchanged (safe + idempotent).
	const unknown = "some upstream passthrough body"
	if got := Translate(LangZH, unknown); got != unknown {
		t.Errorf("Translate(zh, unknown) = %q, want passthrough", got)
	}
	// Empty stays empty.
	if got := Translate(LangZH, ""); got != "" {
		t.Errorf("Translate(zh, empty) = %q", got)
	}
}

func TestTranslatef(t *testing.T) {
	if got := Translatef(LangEN, "No available accounts: %s", "boom"); got != "No available accounts: boom" {
		t.Errorf("Translatef(en) = %q", got)
	}
	if got := Translatef(LangZH, "No available accounts: %s", "boom"); got != "无可用账号：boom" {
		t.Errorf("Translatef(zh) = %q, want 无可用账号：boom", got)
	}
}
