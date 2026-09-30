package auth

import "testing"

func TestSafeRedirect(t *testing.T) {
	const base = ".turkushan.com"
	for rd, want := range map[string]string{
		"https://lrr.turkushan.com/archive?id=1": "https://lrr.turkushan.com/archive?id=1",
		"https://turkushan.com/":                 "https://turkushan.com/",
		"https://a.b.turkushan.com":              "https://a.b.turkushan.com",
		"https://LRR.Turkushan.com/x":            "https://LRR.Turkushan.com/x",
		"":                                       "",
		"http://lrr.turkushan.com/":              "", // plain http not allowed on an https portal
		"https://evil.com/":                      "",
		"https://turkushan.com.evil.com/":        "",
		"https://eviltturkushan.com/":            "",
		"https://evilturkushan.com/":             "",
		"https://turkushan.com@evil.com/":        "",
		"https://user:pw@lrr.turkushan.com/":     "",
		"//evil.com/":                            "",
		"/relative/path":                         "",
		"javascript:alert(1)":                    "",
		"https://lrr.turkushan.com\\@evil.com":   "",
		"https://lrr.turkushan.com/\r\nx":        "",
	} {
		if got := SafeRedirect(rd, base, false); got != want {
			t.Errorf("%q: got %q, want %q", rd, got, want)
		}
	}

	if got := SafeRedirect("http://app.localtest.me/", "localtest.me", true); got == "" {
		t.Error("http should be allowed when the portal itself runs on http")
	}
	if got := SafeRedirect("https://x.turkushan.com/", "", false); got != "" {
		t.Error("empty base domain must reject everything")
	}
}

func TestNormalizeEmail(t *testing.T) {
	for in, want := range map[string]string{
		"":                      "",
		"  ":                    "",
		" Turk@Example.COM ":    "turk@example.com",
		"a.b+tag@mail.example.nl": "a.b+tag@mail.example.nl",
	} {
		got, err := NormalizeEmail(in)
		if err != nil || got != want {
			t.Errorf("%q: got %q %v, want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"nope", "a@b", "Name <a@b.com>", "a@@b.com", "a b@c.com"} {
		if _, err := NormalizeEmail(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}
