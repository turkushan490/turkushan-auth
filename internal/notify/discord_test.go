package notify

import "testing"

func TestValidDiscordWebhook(t *testing.T) {
	for u, want := range map[string]bool{
		"https://discord.com/api/webhooks/123456/abcDEF_-9":        true,
		"https://discordapp.com/api/webhooks/123456/abc":           true,
		"https://canary.discord.com/api/webhooks/1/x":              true,
		"http://discord.com/api/webhooks/123456/abc":               false,
		"https://discord.com.evil.com/api/webhooks/1/x":            false,
		"https://evil.com/?https://discord.com/api/webhooks/1/x":   false,
		"https://discord.com/api/webhooks/123456/abc/../../users":  false,
		"https://192.168.0.1/api/webhooks/1/x":                     false,
		"":                                                         false,
	} {
		if got := ValidDiscordWebhook(u); got != want {
			t.Errorf("%q: got %v, want %v", u, got, want)
		}
	}
}
