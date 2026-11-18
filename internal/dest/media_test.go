package dest

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sgnd/kongkow/internal/store"
)

var pngHeader = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")

func writeFile(t *testing.T, name string, data []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestMastodonUploadsImages(t *testing.T) {
	f := newFakeAPI(t, map[string]string{
		"/api/v2/media":    `{"id":"m1"}`,
		"/api/v1/statuses": `{"url":"https://m.example/@me/2"}`,
	})
	img := writeFile(t, "a.png", pngHeader)
	d := &store.Dest{Name: "m", Kind: store.DestMastodon, Target: f.srv.URL, SecretRef: "secret:m"}
	if _, err := Publish(t.Context(), d, Content{Body: "pic", Media: []string{img, img}}, secretResolver); err != nil {
		t.Fatal(err)
	}
	got := f.got["/api/v1/statuses"]
	if ids, _ := got["media_ids"].([]any); len(ids) != 2 || ids[0] != "m1" {
		t.Fatalf("media_ids %v", got["media_ids"])
	}
	if got["status"] != "pic" {
		t.Fatalf("media paths leaked into the text: %q", got["status"])
	}
	if f.auth["/api/v2/media"] != "Bearer SECRET" {
		t.Fatalf("upload auth %q", f.auth["/api/v2/media"])
	}
}

func TestBlueskyEmbedsImages(t *testing.T) {
	f := newFakeAPI(t, map[string]string{
		"/xrpc/com.atproto.server.createSession": `{"accessJwt":"JWT","did":"did:plc:me","handle":"me"}`,
		"/xrpc/com.atproto.repo.uploadBlob":      `{"blob":{"$type":"blob","ref":{"$link":"bafy"},"mimeType":"image/png","size":16}}`,
		"/xrpc/com.atproto.repo.createRecord":    `{"uri":"at://did:plc:me/app.bsky.feed.post/2"}`,
	})
	old := bskyService
	bskyService = f.srv.URL
	t.Cleanup(func() { bskyService = old })

	img := writeFile(t, "a.png", pngHeader)
	d := &store.Dest{Name: "b", Kind: store.DestBluesky, Target: "me", SecretRef: "secret:b"}
	if _, err := Publish(t.Context(), d, Content{Body: "pic", Media: []string{img}, Alt: []string{"A red logo"}}, secretResolver); err != nil {
		t.Fatal(err)
	}
	record := f.got["/xrpc/com.atproto.repo.createRecord"]["record"].(map[string]any)
	embed, _ := record["embed"].(map[string]any)
	if embed["$type"] != "app.bsky.embed.images" {
		t.Fatalf("embed %v", record["embed"])
	}
	images := embed["images"].([]any)
	blob := images[0].(map[string]any)["image"].(map[string]any)
	if images[0].(map[string]any)["alt"] != "A red logo" {
		t.Fatalf("alt %v", images[0])
	}
	if len(images) != 1 || blob["mimeType"] != "image/png" {
		t.Fatalf("images %v", images)
	}
	if record["text"] != "pic" {
		t.Fatalf("text %q", record["text"])
	}
}

func TestCheckMedia(t *testing.T) {
	img := writeFile(t, "a.png", pngHeader)
	txt := writeFile(t, "a.txt", []byte("hello"))
	big := writeFile(t, "big.png", append(pngHeader, make([]byte, blueskyImageSize)...))
	bs := &store.Dest{Kind: store.DestBluesky}
	m := &store.Dest{Kind: store.DestMastodon}
	cases := []struct {
		d     *store.Dest
		media []string
		want  string
	}{
		{bs, []string{img, img, img, img, img}, "at most 4 images"},
		{m, []string{txt}, "only JPEG, PNG, GIF and WebP"},
		{bs, []string{big}, "takes at most 1000000"},
	}
	for _, c := range cases {
		err := CheckMedia(c.d, Content{Media: c.media})
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s %v: want %q, got %v", c.d.Kind, c.media, c.want, err)
		}
	}
	long := strings.Repeat("a", maxAlt+1)
	if err := CheckMedia(m, Content{Media: []string{img}, Alt: []string{long}}); err == nil {
		t.Fatal("alt text over the limit should be refused")
	}
	if err := CheckMedia(m, Content{Media: []string{t.TempDir()}}); err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("directory as media: %v", err)
	}
	gone := filepath.Join(t.TempDir(), "gone.png")
	if err := CheckMedia(m, Content{Media: []string{gone}}); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing file: %v", err)
	}
	if err := CheckMedia(m, Content{Media: []string{big}}); err != nil {
		t.Fatalf("mastodon takes 1 MB: %v", err)
	}
	// Telegram and file destinations keep listing the paths as text.
	if err := CheckMedia(&store.Dest{Kind: store.DestTelegram}, Content{Media: []string{txt}}); err != nil {
		t.Fatal(err)
	}
	if got := postText(&store.Dest{Kind: store.DestTelegram}, Content{Body: "x", Media: []string{txt}}); !strings.Contains(got, txt) {
		t.Fatalf("telegram text %q", got)
	}
}
