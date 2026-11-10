package server

import (
	"strings"

	"github.com/turkushan490/turkushan-auth/internal/auth"
)

// errorCodes gives the messages visitors can see a stable code. writeError
// sends it next to the English text as {"error": …, "code": …}, and the web app
// shows its own translation of err.<code> when it has one. Messages without a
// code (mostly admin-panel ones) are shown in English.
var errorCodes = map[string]string{
	msgBadLogin: "bad_login",
	"Too many wrong passwords. This account is locked for 15 minutes.": "locked",
	"This account is blocked.":                                                               "blocked",
	"Please sign in.":                                                                        "sign_in",
	"Wrong password.":                                                                        "wrong_password",
	"Your current password is wrong.":                                                        "current_password_wrong",
	"That username is already taken.":                                                        "username_taken",
	"That email address is already used by another account.":                                 "email_taken",
	"Another account already verified this email address.":                                   "email_verified_elsewhere",
	"Email isn't set up on this portal yet. Ask the admin.":                                  "mail_not_ready",
	"The mail could not be sent. Try again later.":                                           "mail_failed",
	"Your address is saved, but the mail could not be sent. Try \"Send again\" later.":       "mail_saved_not_sent",
	"There is no unverified email address to send to.":                                       "no_unverified_email",
	"This link is for an email address you no longer use.":                                   "link_old_email",
	"This link is invalid, already used or expired. Request a new one on your account page.": "link_invalid_account",
	"This link is invalid, already used or expired. Request a new one.":                      "link_invalid",
	"This sign-up has expired. Start again from the sign-in page.":                           "signup_expired",
	"This sign-up is already finished. Sign in from the sign-in page.":                       "signup_finished",
	"Set a password first. Otherwise you can't sign in anymore.":                             "set_password_first",
	"Too many attempts. Wait a minute and try again.":                                        "rate_minute",
	"Too many attempts. Try again later.":                                                    "rate_later",
	"Too many email changes. Try again in an hour.":                                          "rate_email_changes",
	"Too many mails sent. Try again in an hour.":                                             "rate_mails",
	"Too many new accounts from your network. Try again later.":                              "rate_signups",
	"Too many requests. Try again in an hour.":                                               "rate_hour",
	"Security token missing or expired. Reload the page and try again.":                      "csrf",
	"Something went wrong. Try again.":                                                       "server",
	"Invalid request.":                                                                       "invalid_request",
	"You don't have the right to do that.":                                                   "no_right",
}

func init() {
	// Validation errors reach the user through sentence(err).
	for err, code := range map[error]string{
		auth.ErrUsernameInvalid:  "username_invalid",
		auth.ErrPasswordTooShort: "password_short",
		auth.ErrPasswordTooLong:  "password_long",
		auth.ErrPasswordNoUpper:  "password_upper",
		auth.ErrPasswordNoSymbol: "password_symbol",
		auth.ErrEmailInvalid:     "email_invalid",
	} {
		errorCodes[sentence(err)] = code
	}
}

// errorCode returns the code for msg, also for messages that start with a known one.
func errorCode(msg string) string {
	if c, ok := errorCodes[msg]; ok {
		return c
	}
	if strings.HasPrefix(msg, "Request blocked.") {
		return "origin"
	}
	return ""
}
