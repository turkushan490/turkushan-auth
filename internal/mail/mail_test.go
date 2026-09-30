package mail_test

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/turkushan490/turkushan-auth/internal/mail"
	"github.com/turkushan490/turkushan-auth/internal/mail/mailtest"
)

func TestSend(t *testing.T) {
	srv := mailtest.Start(t)
	port, _ := strconv.Atoi(srv.Port)
	cfg := mail.Config{Host: srv.Host, Port: port, Username: "resend", Password: "key", From: "Portal <noreply@example.com>"}

	long := strings.Repeat("a long line with ümlauts ", 10) + "https://auth.example.com/verify?token=abc_DEF-123"
	err := mail.Send(context.Background(), cfg, mail.Message{
		To: "User@Example.com", Subject: "Verify your email ✓", Text: long, HTML: "<p>hi</p>",
	})
	if err != nil {
		t.Fatal(err)
	}
	to, text := srv.Last(t)
	if !strings.Contains(to, "User@Example.com") {
		t.Errorf("to: %q", to)
	}
	if text != long {
		t.Errorf("text part did not survive encoding:\n%q\n%q", text, long)
	}
}

func TestSendRefusesBadInput(t *testing.T) {
	cfg := mail.Config{Host: "127.0.0.1", Port: 1, From: "noreply@example.com"}
	if err := mail.Send(context.Background(), cfg, mail.Message{To: "a@b.com\r\nBcc: x@y.com"}); err == nil {
		t.Error("header injection in To accepted")
	}
	if err := mail.Send(context.Background(), mail.Config{}, mail.Message{To: "a@b.com"}); err == nil {
		t.Error("unconfigured mail accepted")
	}
	for from, ok := range map[string]bool{
		"noreply@turkushan.com":                     true,
		"Turkushan Portal <noreply@turkushan.com>":  true,
		`"turkushan.com" <noreply@turkushan.com>`:   true,
		"not an address":                            false,
		"a@b.com\r\nBcc: x@y.com":                   false,
	} {
		if mail.ValidFrom(from) != ok {
			t.Errorf("ValidFrom(%q) != %v", from, ok)
		}
	}
}
