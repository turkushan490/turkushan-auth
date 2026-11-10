package i18n

import (
	"regexp"
	"testing"
)

func TestFromAcceptLanguage(t *testing.T) {
	for header, want := range map[string]string{
		"nl-NL,nl;q=0.9,en;q=0.8": "nl",
		"pt-BR":                   "pt",
		"zh-Hant-TW,zh;q=0.8":     "zh-TW",
		"zh-CN":                   "zh-CN",
		"zh":                      "zh-CN",
		"nb-NO":                   "no",
		"xx, tr;q=0.5":            "tr",
		"xx":                      "",
		"":                        "",
	} {
		if got := FromAcceptLanguage(header); got != want {
			t.Errorf("%q: got %q, want %q", header, got, want)
		}
	}
}

func TestMailFallbackAndParams(t *testing.T) {
	if got := Mail("en", "verify.intro", map[string]string{"user": "ann", "site": "example.com"}); got != "Hi ann, click the button to confirm this is your email address for example.com." {
		t.Errorf("en: %q", got)
	}
	if got := Mail("xx", "verify.button", nil); got != "Verify email" {
		t.Errorf("unknown language falls back to English: %q", got)
	}
}

// Every translation has the same placeholders as English, so no {user} goes missing.
func TestMailTranslationsKeepPlaceholders(t *testing.T) {
	ph := regexp.MustCompile(`\{[a-z]+\}`)
	for _, lang := range Supported {
		for _, key := range Keys() {
			if !Has(lang, key) {
				continue
			}
			want := ph.FindAllString(Mail("en", key, nil), -1)
			got := Mail(lang, key, nil)
			for _, p := range want {
				if !regexp.MustCompile(regexp.QuoteMeta(p)).MatchString(got) {
					t.Errorf("%s %s lacks %s: %q", lang, key, p, got)
				}
			}
		}
	}
}
