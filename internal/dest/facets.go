package dest

import (
	"context"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// Bluesky shows a link or mention as plain text unless the post carries a
// facet for it. Facet offsets count UTF-8 bytes, not characters.
var (
	linkRe    = regexp.MustCompile(`https?://[^\s<>"]+`)
	mentionRe = regexp.MustCompile(`(^|[^\w@/.])(@((?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z][a-zA-Z0-9-]*))`)
)

type facet struct {
	Index    facetIndex       `json:"index"`
	Features []map[string]any `json:"features"`
}

type facetIndex struct {
	ByteStart int `json:"byteStart"`
	ByteEnd   int `json:"byteEnd"`
}

// bskyFacets finds links and mentions in text. resolve turns a handle into
// a DID; a mention it cannot resolve stays plain text.
func bskyFacets(text string, resolve func(handle string) (string, error)) []facet {
	var fs []facet
	var links [][2]int
	for _, m := range linkRe.FindAllStringIndex(text, -1) {
		// a link at the end of a sentence keeps its full stop out
		end := m[0] + len(strings.TrimRight(text[m[0]:m[1]], ".,;:!?)'"))
		links = append(links, [2]int{m[0], end})
		fs = append(fs, facet{
			Index:    facetIndex{m[0], end},
			Features: []map[string]any{{"$type": "app.bsky.richtext.facet#link", "uri": text[m[0]:end]}},
		})
	}
	for _, m := range mentionRe.FindAllStringSubmatchIndex(text, -1) {
		start, end := m[4], m[5]
		inLink := false
		for _, l := range links {
			if start < l[1] && end > l[0] {
				inLink = true
			}
		}
		if inLink {
			continue
		}
		did, err := resolve(text[m[6]:m[7]])
		if err != nil || did == "" {
			continue
		}
		fs = append(fs, facet{
			Index:    facetIndex{start, end},
			Features: []map[string]any{{"$type": "app.bsky.richtext.facet#mention", "did": did}},
		})
	}
	return fs
}

func bskyResolveHandle(ctx context.Context, handle string) (string, error) {
	var res struct {
		DID string `json:"did"`
	}
	u := bskyService + "/xrpc/com.atproto.identity.resolveHandle?handle=" + url.QueryEscape(handle)
	if err := jsonCall(ctx, http.MethodGet, u, "", nil, &res); err != nil {
		return "", err
	}
	return res.DID, nil
}
