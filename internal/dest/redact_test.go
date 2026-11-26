package dest

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sgnd/kongkow/internal/store"
)

// fakeToken has the shape of a Telegram bot token and is not one.
const fakeToken = "123456789:AAFAKEFAKEFAKEFAKEFAKEFAKEFAKEFAKE_-x"

func tokenResolverFake(string) (string, error) { return fakeToken, nil }

func TestRedact(t *testing.T) {
	cases := []struct{ in, want string }{
		{`Post "https://api.telegram.org/bot` + fakeToken + `/sendMessage": dial tcp: refused`, `Post "https://api.telegram.org/bot<redacted>/sendMessage": dial tcp: refused`},
		{"no token here", "no token here"},
		{"/bot" + fakeToken + "/getMe and /bot" + fakeToken + "/sendMessage", "/bot<redacted>/getMe and /bot<redacted>/sendMessage"},
		{"a /bot-like path /bot/x", "a /bot-like path /bot/x"},
	}
	for _, c := range cases {
		if got := Redact(c.in); got != c.want {
			t.Errorf("Redact(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// A network failure must not put the bot token in the error: Go prints the
// whole URL of a failed request, and the token is part of Telegram's URL.
func TestTelegramNetworkErrorHasNoToken(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	telegramBaseWas := telegramBase
	telegramBase = srv.URL
	srv.Close() // nothing listens there now
	t.Cleanup(func() { telegramBase = telegramBaseWas })

	d := tgDest()
	_, err := Publish(t.Context(), d, Content{Name: "n", Body: "x"}, tokenResolverFake)
	if err == nil {
		t.Fatal("a closed server must fail")
	}
	if strings.Contains(err.Error(), fakeToken) || strings.Contains(err.Error(), "AAFAKE") {
		t.Fatalf("publish error holds the token: %v", err)
	}
	if !strings.Contains(err.Error(), "/bot<redacted>/sendMessage") {
		t.Errorf("error should still say which call failed: %v", err)
	}
	_, err = Test(t.Context(), d, tokenResolverFake)
	if err == nil || strings.Contains(err.Error(), "AAFAKE") {
		t.Fatalf("test error: %v", err)
	}
}

func TestOtherDestinationErrorsHaveNoToken(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	base := srv.URL
	srv.Close()
	old := bskyService
	bskyService = base
	t.Cleanup(func() { bskyService = old })

	for _, d := range []*store.Dest{
		{Name: "m", Kind: store.DestMastodon, Target: base, SecretRef: "secret:m"},
		{Name: "b", Kind: store.DestBluesky, Target: "me.bsky.social", SecretRef: "secret:b"},
	} {
		_, err := Publish(t.Context(), d, Content{Body: "x"}, tokenResolverFake)
		if err == nil {
			t.Fatalf("%s: a closed server must fail", d.Kind)
		}
		if strings.Contains(err.Error(), "AAFAKE") {
			t.Errorf("%s error holds the token: %v", d.Kind, err)
		}
	}
}

func TestMastodonURLMustBeHTTPSOnTheDestinationsHost(t *testing.T) {
	cases := []struct{ name, url, want string }{
		{"same host", "https://HOST/@me/1", "https://HOST/@me/1"},
		{"other host", "https://evil.example/@me/1", ""},
		{"plain http", "http://HOST/@me/1", ""},
		{"script", "javascript:alert(1)", ""},
		{"with credentials", "https://user:pw@HOST/@me/1", ""},
		{"empty", "", ""},
	}
	for _, c := range cases {
		replies := map[string]string{}
		f := newFakeAPI(t, replies)
		host := strings.TrimPrefix(f.srv.URL, "http://")
		url := strings.ReplaceAll(c.url, "HOST", host)
		replies["/api/v1/statuses"] = `{"url":"` + url + `"}`
		d := &store.Dest{Name: "m", Kind: store.DestMastodon, Target: f.srv.URL, SecretRef: "secret:m"}
		r, err := Publish(t.Context(), d, Content{Body: "x"}, secretResolver)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		want := strings.ReplaceAll(c.want, "HOST", host)
		if r.URL != want {
			t.Errorf("%s: url %q, want %q", c.name, r.URL, want)
		}
		if want == "" && strings.Contains(r.Detail, "evil.example") {
			t.Errorf("%s: the refused address is still in the detail: %q", c.name, r.Detail)
		}
	}
}
