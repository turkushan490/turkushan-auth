// Package i18n knows the supported languages and holds the translated mail texts.
// The web app has its own, much larger, translations in web/src/locales.
package i18n

import (
	"embed"
	"encoding/json"
	"strings"
	"sync"
)

// Supported lists every language the portal can show, in the same codes as
// web/src/locales/index.js. Keep both lists the same.
var Supported = []string{
	"en", "nl", "tr", "de", "fr", "es", "pt", "it", "pl", "ru", "uk", "ar", "fa",
	"hi", "ja", "ko", "zh-CN", "zh-TW", "id", "vi", "th", "sv", "da", "no", "fi",
}

var supported = func() map[string]string {
	m := map[string]string{}
	for _, c := range Supported {
		m[strings.ToLower(c)] = c
	}
	return m
}()

// Canonical returns the supported spelling of code ("zh-cn" → "zh-CN"), or "" when unsupported.
func Canonical(code string) string {
	return supported[strings.ToLower(strings.TrimSpace(code))]
}

// FromAcceptLanguage picks the best supported language from an Accept-Language
// header, or "" when none fits. "pt-BR" falls back to "pt", "zh-Hant" to "zh-TW".
func FromAcceptLanguage(header string) string {
	for _, part := range strings.Split(header, ",") {
		tag := strings.TrimSpace(strings.SplitN(part, ";", 2)[0])
		if c := Canonical(tag); c != "" {
			return c
		}
		lower := strings.ToLower(tag)
		switch {
		case strings.HasPrefix(lower, "zh-hant"), lower == "zh-hk", lower == "zh-mo":
			return "zh-TW"
		case strings.HasPrefix(lower, "zh"):
			return "zh-CN"
		case strings.HasPrefix(lower, "nb"), strings.HasPrefix(lower, "nn"):
			return "no"
		}
		if base, _, ok := strings.Cut(lower, "-"); ok {
			if c := Canonical(base); c != "" {
				return c
			}
		}
	}
	return ""
}

// RTL reports whether the language is written right to left.
func RTL(lang string) bool { return lang == "ar" || lang == "fa" }

//go:embed mail/*.json
var mailFiles embed.FS

var (
	mailOnce sync.Once
	mailText map[string]map[string]string // lang → key → text
)

func loadMail() {
	mailText = map[string]map[string]string{}
	for _, lang := range Supported {
		b, err := mailFiles.ReadFile("mail/" + lang + ".json")
		if err != nil {
			continue // not translated yet: English is used
		}
		var m map[string]string
		if err := json.Unmarshal(b, &m); err == nil {
			mailText[lang] = m
		}
	}
}

// Mail returns the mail text for key in lang (English when missing), with
// {name} placeholders filled from params.
func Mail(lang, key string, params map[string]string) string {
	mailOnce.Do(loadMail)
	s, ok := mailText[lang][key]
	if !ok || s == "" {
		s, ok = mailText["en"][key]
	}
	if !ok {
		s = key
	}
	for k, v := range params {
		s = strings.ReplaceAll(s, "{"+k+"}", v)
	}
	return s
}

// Keys returns the English mail keys, for tests that check every translation.
func Keys() []string {
	mailOnce.Do(loadMail)
	out := make([]string, 0, len(mailText["en"]))
	for k := range mailText["en"] {
		out = append(out, k)
	}
	return out
}

// Has reports whether lang has its own text for key.
func Has(lang, key string) bool {
	mailOnce.Do(loadMail)
	_, ok := mailText[lang][key]
	return ok
}
