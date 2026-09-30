// Package notify sends admin notifications.
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"
)

var webhookRe = regexp.MustCompile(`^https://(?:ptb\.|canary\.)?discord(?:app)?\.com/api/webhooks/\d+/[A-Za-z0-9_-]+$`)

// ValidDiscordWebhook only accepts real Discord webhook URLs, so the setting
// cannot be used to make the portal call arbitrary addresses.
func ValidDiscordWebhook(u string) bool {
	return webhookRe.MatchString(u)
}

var client = &http.Client{Timeout: 10 * time.Second}

// Discord posts one embed to a webhook.
func Discord(ctx context.Context, webhook, title, description, link string) error {
	if !ValidDiscordWebhook(webhook) {
		return fmt.Errorf("not a Discord webhook URL")
	}
	embed := map[string]any{"title": title, "description": description, "color": 0x6366f1}
	if link != "" {
		embed["url"] = link
	}
	body, _ := json.Marshal(map[string]any{"embeds": []any{embed}})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("discord answered %s", resp.Status)
	}
	return nil
}
