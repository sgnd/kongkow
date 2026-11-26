package dest

import (
	"errors"
	"net/url"

	"github.com/sgnd/kongkow/internal/redact"
)

// Redact hides a Telegram bot token in s (see redact.Token).
func Redact(s string) string { return redact.Token(s) }

// scrubErr removes a token from the URL that a failed request's error carries.
func scrubErr(err error) error {
	var ue *url.Error
	if errors.As(err, &ue) {
		ue.URL = Redact(ue.URL)
	}
	return err
}

// sameHostHTTPS returns link when it is an https address on the same host as
// base, and "" otherwise: a server's answer must not send people elsewhere.
func sameHostHTTPS(base, link string) string {
	b, err1 := url.Parse(base)
	l, err2 := url.Parse(link)
	if err1 != nil || err2 != nil || l.Scheme != "https" || l.User != nil || l.Host == "" || l.Host != b.Host {
		return ""
	}
	return link
}
