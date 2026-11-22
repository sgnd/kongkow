package dest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sgnd/kongkow/internal/store"
)

// fakeAPI answers each path with a fixed JSON reply and records the
// requests it got, keyed by path.
type fakeAPI struct {
	srv  *httptest.Server
	got  map[string]map[string]any
	auth map[string]string
}

func newFakeAPI(t *testing.T, replies map[string]string) *fakeAPI {
	t.Helper()
	f := &fakeAPI{got: map[string]map[string]any{}, auth: map[string]string{}}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reply, ok := replies[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"not found"}`))
			return
		}
		body := map[string]any{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.got[r.URL.Path] = body
		f.auth[r.URL.Path] = r.Header.Get("Authorization")
		if strings.HasPrefix(reply, "!") { // an error answer
			w.WriteHeader(http.StatusUnauthorized)
			reply = reply[1:]
		}
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func secretResolver(ref string) (string, error) { return "SECRET", nil }

func TestMastodonPublish(t *testing.T) {
	replies := map[string]string{}
	f := newFakeAPI(t, replies)
	link := "https://" + strings.TrimPrefix(f.srv.URL, "http://") + "/@me/1"
	replies["/api/v1/statuses"] = `{"url":"` + link + `"}`
	d := &store.Dest{Name: "m", Kind: store.DestMastodon, Target: f.srv.URL, SecretRef: "secret:m"}
	r, err := Publish(t.Context(), d, Content{Title: "Hi", Body: "world"}, secretResolver)
	if err != nil {
		t.Fatal(err)
	}
	if r.Detail != "posted "+link {
		t.Fatalf("detail %q", r.Detail)
	}
	if f.got["/api/v1/statuses"]["status"] != "Hi\n\nworld" || f.auth["/api/v1/statuses"] != "Bearer SECRET" {
		t.Fatalf("request %v %q", f.got, f.auth)
	}
}

func TestMastodonTestAndError(t *testing.T) {
	f := newFakeAPI(t, map[string]string{
		"/api/v1/accounts/verify_credentials": `{"acct":"me"}`,
		"/api/v1/statuses":                    `!{"error":"The access token is invalid"}`,
	})
	d := &store.Dest{Name: "m", Kind: store.DestMastodon, Target: f.srv.URL, SecretRef: "secret:m"}
	r, err := Test(t.Context(), d, secretResolver)
	if err != nil || r.Detail != "account @me" {
		t.Fatalf("test %v %v", r, err)
	}
	_, err = Publish(t.Context(), d, Content{Body: "x"}, secretResolver)
	if err == nil || !strings.Contains(err.Error(), "access token is invalid") {
		t.Fatalf("err %v", err)
	}
}

func TestMastodonNeedsHTTPS(t *testing.T) {
	for _, target := range []string{"http://mastodon.social", "mastodon.social", ""} {
		if _, err := MastodonBase(target); err == nil {
			t.Fatalf("%q should be refused", target)
		}
	}
	if b, err := MastodonBase("https://mastodon.social/"); err != nil || b != "https://mastodon.social" {
		t.Fatalf("%q %v", b, err)
	}
}

func TestBlueskyPublish(t *testing.T) {
	f := newFakeAPI(t, map[string]string{
		"/xrpc/com.atproto.server.createSession":   `{"accessJwt":"JWT","did":"did:plc:me","handle":"me.bsky.social"}`,
		"/xrpc/com.atproto.repo.createRecord":      `{"uri":"at://did:plc:me/app.bsky.feed.post/1"}`,
		"/xrpc/com.atproto.identity.resolveHandle": `{"did":"did:plc:friend"}`,
	})
	old := bskyService
	bskyService = f.srv.URL
	t.Cleanup(func() { bskyService = old })

	d := &store.Dest{Name: "b", Kind: store.DestBluesky, Target: "me.bsky.social", SecretRef: "secret:b"}
	r, err := Publish(t.Context(), d, Content{Body: "hello @friend.bsky.social"}, secretResolver)
	if err != nil {
		t.Fatal(err)
	}
	if r.Detail != "posted at://did:plc:me/app.bsky.feed.post/1" {
		t.Fatalf("detail %q", r.Detail)
	}
	login := f.got["/xrpc/com.atproto.server.createSession"]
	if login["identifier"] != "me.bsky.social" || login["password"] != "SECRET" {
		t.Fatalf("login %v", login)
	}
	rec := f.got["/xrpc/com.atproto.repo.createRecord"]
	if rec["repo"] != "did:plc:me" || f.auth["/xrpc/com.atproto.repo.createRecord"] != "Bearer JWT" {
		t.Fatalf("record %v", rec)
	}
	record := rec["record"].(map[string]any)
	if record["text"] != "hello @friend.bsky.social" {
		t.Fatalf("text %v", record["text"])
	}
	if fs, _ := record["facets"].([]any); len(fs) != 1 {
		t.Fatalf("facets %v", record["facets"])
	}
	tr, err := Test(t.Context(), d, secretResolver)
	if err != nil || tr.Detail != "account @me.bsky.social" {
		t.Fatalf("test %v %v", tr, err)
	}
}

func TestLengthLimits(t *testing.T) {
	long := strings.Repeat("é", 301) // runes, not bytes, are counted
	b := &store.Dest{Kind: store.DestBluesky}
	if err := CheckLength(b, Content{Body: long}); err == nil {
		t.Fatal("bluesky should refuse 301 characters")
	}
	if err := CheckLength(b, Content{Body: long[:len(long)-2]}); err != nil {
		t.Fatalf("300 characters should pass: %v", err)
	}
	m := &store.Dest{Kind: store.DestMastodon}
	if err := CheckLength(m, Content{Body: long}); err != nil {
		t.Fatalf("mastodon allows 500: %v", err)
	}
	// Nothing is sent when the post is too long.
	if _, err := Publish(t.Context(), &store.Dest{Kind: store.DestBluesky, Target: "x", SecretRef: "secret:b"},
		Content{Body: long}, secretResolver); err == nil || !strings.Contains(err.Error(), "allows 300") {
		t.Fatalf("err %v", err)
	}
}
