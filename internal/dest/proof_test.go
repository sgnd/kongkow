package dest

import (
	"testing"

	"github.com/sgnd/kongkow/internal/store"
)

// Publish proof: every result that has a public page names it in URL.

func TestMastodonResultHasURL(t *testing.T) {
	f := newFakeAPI(t, map[string]string{"/api/v1/statuses": `{"url":"https://m.example/@me/1"}`})
	d := &store.Dest{Name: "m", Kind: store.DestMastodon, Target: f.srv.URL, SecretRef: "secret:m"}
	r, err := Publish(t.Context(), d, Content{Body: "x"}, secretResolver)
	if err != nil {
		t.Fatal(err)
	}
	if r.URL != "https://m.example/@me/1" {
		t.Errorf("url %q", r.URL)
	}
}

func TestBlueskyResultHasURL(t *testing.T) {
	f := newFakeAPI(t, map[string]string{
		"/xrpc/com.atproto.server.createSession": `{"accessJwt":"JWT","did":"did:plc:me","handle":"me.bsky.social"}`,
		"/xrpc/com.atproto.repo.createRecord":    `{"uri":"at://did:plc:me/app.bsky.feed.post/3kabc"}`,
	})
	old := bskyService
	bskyService = f.srv.URL
	t.Cleanup(func() { bskyService = old })
	d := &store.Dest{Name: "b", Kind: store.DestBluesky, Target: "me.bsky.social", SecretRef: "secret:b"}
	r, err := Publish(t.Context(), d, Content{Body: "x"}, secretResolver)
	if err != nil {
		t.Fatal(err)
	}
	if want := "https://bsky.app/profile/me.bsky.social/post/3kabc"; r.URL != want {
		t.Errorf("url %q, want %q", r.URL, want)
	}
}

func TestBlueskyURLFromURI(t *testing.T) {
	cases := []struct{ uri, who, want string }{
		{"at://did:plc:me/app.bsky.feed.post/3k", "me.bsky.social", "https://bsky.app/profile/me.bsky.social/post/3k"},
		{"at://did:plc:me/app.bsky.feed.post/3k", "", "https://bsky.app/profile/did:plc:me/post/3k"},
		{"at://did:plc:me/app.bsky.feed.like/3k", "me", ""}, // not a post
		{"https://evil.example/x", "me", ""},
		{"", "me", ""},
	}
	for _, c := range cases {
		if got := blueskyURL(c.uri, c.who); got != c.want {
			t.Errorf("blueskyURL(%q, %q) = %q, want %q", c.uri, c.who, got, c.want)
		}
	}
}

func TestTelegramResultURLOnlyForPublicChannels(t *testing.T) {
	fakeTelegram(t, `{"ok":true,"result":{"message_id":7,"chat":{"username":"mychan"}}}`)
	r, err := Publish(t.Context(), tgDest(), Content{Name: "n", Body: "x"}, tokenResolver)
	if err != nil {
		t.Fatal(err)
	}
	if r.URL != "https://t.me/mychan/7" {
		t.Errorf("public channel url %q", r.URL)
	}

	fakeTelegram(t, `{"ok":true,"result":{"message_id":8,"chat":{"id":42}}}`)
	r, err = Publish(t.Context(), tgDest(), Content{Name: "n", Body: "x"}, tokenResolver)
	if err != nil {
		t.Fatal(err)
	}
	if r.URL != "" {
		t.Errorf("a private chat has no public page, got %q", r.URL)
	}
	if r.Detail != "sent message 8" {
		t.Errorf("detail %q: the message id is the proof for a private chat", r.Detail)
	}
}

func TestFileResultHasNoURLButAPath(t *testing.T) {
	d := &store.Dest{Name: "f", Kind: store.DestFile, Target: t.TempDir()}
	r, err := Publish(t.Context(), d, Content{Name: "n", Body: "x"}, tokenResolver)
	if err != nil {
		t.Fatal(err)
	}
	if r.URL != "" || r.Path == "" {
		t.Errorf("url %q path %q", r.URL, r.Path)
	}
}
