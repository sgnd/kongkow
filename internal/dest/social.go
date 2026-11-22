package dest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/sgnd/kongkow/internal/store"
)

// Text limits per network, counted in characters (runes).
const (
	mastodonLimit = 500
	blueskyLimit  = 300
)

// bskyService is the Bluesky PDS address; tests point it at a fake server.
var bskyService = "https://bsky.social"

// postText is the text sent to a network: title, body, then media paths,
// except where the network gets the images themselves.
func postText(d *store.Dest, c Content) string {
	text := c.Body
	if c.Title != "" {
		text = c.Title + "\n\n" + c.Body
	}
	if len(c.Media) > 0 && !uploadsMedia(d) {
		text += "\n\nMedia:\n" + strings.Join(c.Media, "\n")
	}
	if strings.TrimSpace(text) == "" {
		text = c.Name
	}
	return text
}

// CheckLength says whether c fits the text limit of d's network.
func CheckLength(d *store.Dest, c Content) error {
	limit := 0
	switch d.Kind {
	case store.DestMastodon:
		limit = mastodonLimit
	case store.DestBluesky:
		limit = blueskyLimit
	}
	if n := utf8.RuneCountInString(postText(d, c)); limit > 0 && n > limit {
		return fmt.Errorf("%s allows %d characters; this post has %d", d.Kind, limit, n)
	}
	return nil
}

func needToken(d *store.Dest, resolve ResolveToken, what string) (string, error) {
	if d.Target == "" {
		return "", fmt.Errorf("%s destination needs --target", d.Kind)
	}
	if d.SecretRef == "" {
		return "", fmt.Errorf("%s destination needs --secret-ref secret:NAME for the %s", d.Kind, what)
	}
	tok, err := resolve(d.SecretRef)
	if err != nil {
		return "", err
	}
	if tok == "" {
		return "", fmt.Errorf("%s %s is empty", d.Kind, what)
	}
	return tok, nil
}

// MastodonBase checks a Mastodon instance address: https, or http only on
// this computer.
func MastodonBase(target string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(target))
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("mastodon --target must be the instance address, like https://mastodon.social")
	}
	local := u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1"
	if u.Scheme != "https" && !(u.Scheme == "http" && local) {
		return "", fmt.Errorf("mastodon --target must use https")
	}
	return u.Scheme + "://" + u.Host, nil
}

func publishMastodon(ctx context.Context, d *store.Dest, c Content, resolve ResolveToken) (*Result, error) {
	if err := CheckLength(d, c); err != nil {
		return nil, err
	}
	base, err := MastodonBase(d.Target)
	if err != nil {
		return nil, err
	}
	imgs, err := loadImages(d, c)
	if err != nil {
		return nil, err
	}
	tok, err := needToken(d, resolve, "access token")
	if err != nil {
		return nil, err
	}
	var res struct {
		URL string `json:"url"`
	}
	payload := map[string]any{"status": postText(d, c)}
	var ids []string
	for _, img := range imgs {
		id, err := mastodonUpload(ctx, base, tok, img)
		if err != nil {
			return nil, fmt.Errorf("mastodon: %w", err)
		}
		ids = append(ids, id)
	}
	if len(ids) > 0 {
		payload["media_ids"] = ids
	}
	if v := d.Options["visibility"]; v != "" {
		payload["visibility"] = v
	}
	if err := jsonCall(ctx, http.MethodPost, base+"/api/v1/statuses", tok, payload, &res); err != nil {
		return nil, fmt.Errorf("mastodon: %w", err)
	}
	return &Result{Dest: d.Name, Kind: d.Kind, Target: base, URL: res.URL, Detail: "posted " + res.URL}, nil
}

func testMastodon(ctx context.Context, d *store.Dest, resolve ResolveToken) (*Result, error) {
	base, err := MastodonBase(d.Target)
	if err != nil {
		return nil, err
	}
	tok, err := needToken(d, resolve, "access token")
	if err != nil {
		return nil, err
	}
	var me struct {
		Acct string `json:"acct"`
	}
	if err := jsonCall(ctx, http.MethodGet, base+"/api/v1/accounts/verify_credentials", tok, nil, &me); err != nil {
		return nil, fmt.Errorf("mastodon: %w", err)
	}
	return &Result{Dest: d.Name, Kind: d.Kind, Target: base, Detail: "account @" + me.Acct}, nil
}

type bskySession struct {
	AccessJwt string `json:"accessJwt"`
	DID       string `json:"did"`
	Handle    string `json:"handle"`
}

func bskyLogin(ctx context.Context, d *store.Dest, resolve ResolveToken) (*bskySession, error) {
	pass, err := needToken(d, resolve, "app password")
	if err != nil {
		return nil, err
	}
	var s bskySession
	body := map[string]any{"identifier": d.Target, "password": pass}
	if err := jsonCall(ctx, http.MethodPost, bskyService+"/xrpc/com.atproto.server.createSession", "", body, &s); err != nil {
		return nil, fmt.Errorf("bluesky: %w", err)
	}
	return &s, nil
}

func publishBluesky(ctx context.Context, d *store.Dest, c Content, resolve ResolveToken) (*Result, error) {
	if err := CheckLength(d, c); err != nil {
		return nil, err
	}
	imgs, err := loadImages(d, c)
	if err != nil {
		return nil, err
	}
	s, err := bskyLogin(ctx, d, resolve)
	if err != nil {
		return nil, err
	}
	var res struct {
		URI string `json:"uri"`
	}
	text := postText(d, c)
	record := map[string]any{
		"$type":     "app.bsky.feed.post",
		"text":      text,
		"createdAt": time.Now().UTC().Format(time.RFC3339),
	}
	if fs := bskyFacets(text, func(h string) (string, error) { return bskyResolveHandle(ctx, h) }); len(fs) > 0 {
		record["facets"] = fs
	}
	if len(imgs) > 0 {
		var embeds []map[string]any
		for _, img := range imgs {
			blob, err := bskyUpload(ctx, s.AccessJwt, img)
			if err != nil {
				return nil, fmt.Errorf("bluesky: %w", err)
			}
			embeds = append(embeds, map[string]any{"alt": img.Alt, "image": blob})
		}
		record["embed"] = map[string]any{"$type": "app.bsky.embed.images", "images": embeds}
	}
	body := map[string]any{"repo": s.DID, "collection": "app.bsky.feed.post", "record": record}
	if err := jsonCall(ctx, http.MethodPost, bskyService+"/xrpc/com.atproto.repo.createRecord", s.AccessJwt, body, &res); err != nil {
		return nil, fmt.Errorf("bluesky: %w", err)
	}
	return &Result{Dest: d.Name, Kind: d.Kind, Target: d.Target, URL: blueskyURL(res.URI, s.Handle), Detail: "posted " + res.URI}, nil
}

func testBluesky(ctx context.Context, d *store.Dest, resolve ResolveToken) (*Result, error) {
	s, err := bskyLogin(ctx, d, resolve)
	if err != nil {
		return nil, err
	}
	return &Result{Dest: d.Name, Kind: d.Kind, Target: d.Target, Detail: "account @" + s.Handle}, nil
}

// jsonCall sends JSON (when body is set) with an optional bearer token and
// reads a JSON answer into out. A non-2xx answer becomes an error with the
// server's message.
func jsonCall(ctx context.Context, method, u, bearer string, body, out any) error {
	if body == nil {
		return rawCall(ctx, method, u, bearer, "", nil, out)
	}
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return rawCall(ctx, method, u, bearer, "application/json", bytes.NewReader(b), out)
}

// rawCall sends body as contentType and reads a JSON answer into out.
func rawCall(ctx context.Context, method, u, bearer, contentType string, body io.Reader, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode/100 != 2 {
		var e struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		_ = json.Unmarshal(raw, &e)
		msg := strings.TrimSpace(e.Message + " " + e.Error)
		if msg == "" {
			msg = http.StatusText(resp.StatusCode)
		}
		return fmt.Errorf("%s (HTTP %d)", msg, resp.StatusCode)
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("could not read answer (%w)", err)
		}
	}
	return nil
}

// blueskyURL is the web address of a post from its at:// URI. who is the
// account's handle; without one the DID in the URI stands in. Anything that
// is not an at:// post URI gives "".
func blueskyURL(uri, who string) string {
	rest, ok := strings.CutPrefix(uri, "at://")
	if !ok {
		return ""
	}
	parts := strings.Split(rest, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] != "app.bsky.feed.post" || parts[2] == "" {
		return ""
	}
	if who == "" {
		who = parts[0]
	}
	return "https://bsky.app/profile/" + who + "/post/" + parts[2]
}
