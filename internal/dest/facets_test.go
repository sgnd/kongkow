package dest

import (
	"errors"
	"testing"
)

func TestBskyFacets(t *testing.T) {
	text := "Héllo @alice.bsky.social, see https://example.com/a?b=1. Mail bob@example.com or @ghost.example"
	resolve := func(h string) (string, error) {
		if h == "alice.bsky.social" {
			return "did:plc:alice", nil
		}
		return "", errors.New("unknown handle")
	}
	fs := bskyFacets(text, resolve)
	if len(fs) != 2 {
		t.Fatalf("want 2 facets (link, alice), got %+v", fs)
	}
	link, mention := fs[0], fs[1]
	if got := text[link.Index.ByteStart:link.Index.ByteEnd]; got != "https://example.com/a?b=1" {
		t.Fatalf("link span %q", got)
	}
	if link.Features[0]["uri"] != "https://example.com/a?b=1" {
		t.Fatalf("link %v", link.Features)
	}
	// "é" is two bytes, so the mention starts at byte 7, not character 6.
	if mention.Index.ByteStart != 7 || text[mention.Index.ByteStart:mention.Index.ByteEnd] != "@alice.bsky.social" {
		t.Fatalf("mention %+v", mention.Index)
	}
	if mention.Features[0]["did"] != "did:plc:alice" {
		t.Fatalf("mention %v", mention.Features)
	}
}

func TestBskyFacetsMentionInsideLink(t *testing.T) {
	fs := bskyFacets("https://x.example/@alice.bsky.social", func(string) (string, error) { return "did:plc:a", nil })
	if len(fs) != 1 || fs[0].Features[0]["$type"] != "app.bsky.richtext.facet#link" {
		t.Fatalf("want only the link, got %+v", fs)
	}
}
