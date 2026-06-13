// Package i18n provides lightweight internationalization for gateway error
// messages returned to API clients (e.g. Claude Code / Codex / SDKs).
//
// The language is resolved per-request from the Accept-Language header, falling
// back to the server-configured default language. Only the messages a gateway
// surfaces to API callers are translated; internal logs stay in their original
// (English) form.
package i18n

import (
	"fmt"
	"strings"
	"sync/atomic"
)

// Lang is a supported UI language tag.
type Lang string

const (
	// LangEN is English (the canonical source language for message keys).
	LangEN Lang = "en"
	// LangZH is Simplified Chinese.
	LangZH Lang = "zh"
)

// defaultLang holds the server-configured fallback language. It is read on every
// request, so it is stored in an atomic.Value to stay race-free.
var defaultLang atomic.Value // stores Lang

func init() {
	defaultLang.Store(LangEN)
}

// SetDefault sets the server-wide fallback language used when a request carries
// no usable Accept-Language preference. Unknown values fall back to English.
func SetDefault(lang Lang) {
	defaultLang.Store(ParseLang(string(lang)))
}

// Default returns the configured fallback language.
func Default() Lang {
	if v, ok := defaultLang.Load().(Lang); ok {
		return v
	}
	return LangEN
}

// ParseLang normalizes an arbitrary language string into a supported Lang.
// Anything that is not recognizably Chinese is treated as English.
func ParseLang(s string) Lang {
	s = strings.ToLower(strings.TrimSpace(s))
	switch {
	case s == "":
		return LangEN
	case strings.HasPrefix(s, "zh"):
		return LangZH
	default:
		return LangEN
	}
}

// ResolveAcceptLanguage picks the best supported language from an HTTP
// Accept-Language header value. It honors the order of the listed tags (q-values
// are ignored for simplicity — clients overwhelmingly list their preferred tag
// first). When no supported tag is present it returns the configured default.
func ResolveAcceptLanguage(header string) Lang {
	header = strings.TrimSpace(header)
	if header == "" {
		return Default()
	}
	for _, part := range strings.Split(header, ",") {
		tag := part
		if idx := strings.IndexByte(tag, ';'); idx >= 0 {
			tag = tag[:idx] // drop the ";q=..." weight
		}
		tag = strings.ToLower(strings.TrimSpace(tag))
		switch {
		case tag == "":
			continue
		case strings.HasPrefix(tag, "zh"):
			return LangZH
		case strings.HasPrefix(tag, "en"):
			return LangEN
		}
	}
	return Default()
}

// Translate returns the localized form of an English source message for the
// given language. Unknown messages (e.g. upstream-passthrough or dynamically
// composed strings) are returned unchanged, so calling Translate is always safe
// and idempotent.
func Translate(lang Lang, msg string) string {
	if msg == "" || lang == LangEN {
		return msg
	}
	if table, ok := catalog[lang]; ok {
		if translated, ok := table[msg]; ok {
			return translated
		}
	}
	return msg
}

// Translatef localizes a format string (the catalog key keeps the verbs, e.g.
// "No available accounts: %s") and then applies fmt.Sprintf with the supplied
// arguments. Use it for messages that embed dynamic, non-translatable detail
// such as an underlying error or a size limit.
func Translatef(lang Lang, format string, args ...any) string {
	return fmt.Sprintf(Translate(lang, format), args...)
}
