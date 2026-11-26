// Package redact hides secrets in text that may come from an error.
package redact

import "regexp"

// botToken matches the token part of a Telegram bot URL: /bot<digits>:<secret>.
var botToken = regexp.MustCompile(`(/bot)[0-9]+:[A-Za-z0-9_-]+`)

// Token hides a Telegram bot token in s. Telegram puts the token in the URL
// path, and Go prints the whole URL of a failed request, so any text that
// came from an error may hold one.
func Token(s string) string {
	return botToken.ReplaceAllString(s, "${1}<redacted>")
}
