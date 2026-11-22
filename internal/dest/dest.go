// Package dest publishes posts to Telegram, Mastodon, Bluesky and file sinks.
package dest

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sgnd/kongkow/internal/store"
)

// Content is what gets published.
type Content struct {
	Name  string
	Title string
	Body  string
	Media []string
	Alt   []string // alt text per media file, same order
}

// Result is one publish outcome.
type Result struct {
	Dest   string `json:"dest"`
	Kind   string `json:"kind"`
	Target string `json:"target,omitempty"`
	Path   string `json:"path,omitempty"`
	URL    string `json:"url,omitempty"` // the live post, when it has a public page
	Detail string `json:"detail,omitempty"`
}

// ResolveToken turns a secret reference into a token. The CLI passes vault.Resolve.
type ResolveToken func(ref string) (string, error)

// Publish sends content to d. resolve is used for secret: refs.
func Publish(ctx context.Context, d *store.Dest, c Content, resolve ResolveToken) (*Result, error) {
	switch d.Kind {
	case store.DestFile:
		return publishFile(d, c)
	case store.DestTelegram:
		return publishTelegram(ctx, d, c, resolve)
	case store.DestMastodon:
		return publishMastodon(ctx, d, c, resolve)
	case store.DestBluesky:
		return publishBluesky(ctx, d, c, resolve)
	default:
		return nil, fmt.Errorf("destination kind %q is not supported", d.Kind)
	}
}

// Test checks that a destination looks usable without publishing content.
func Test(ctx context.Context, d *store.Dest, resolve ResolveToken) (*Result, error) {
	switch d.Kind {
	case store.DestFile:
		dir := strings.TrimSpace(d.Target)
		if dir == "" {
			return nil, fmt.Errorf("file destination needs --target DIR")
		}
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, err
		}
		fi, err := os.Stat(dir)
		if err != nil {
			return nil, err
		}
		if !fi.IsDir() {
			return nil, fmt.Errorf("%s is not a directory", dir)
		}
		return &Result{Dest: d.Name, Kind: d.Kind, Target: dir, Detail: "directory writable"}, nil
	case store.DestTelegram:
		if d.Target == "" {
			return nil, fmt.Errorf("telegram destination needs --target CHAT_ID")
		}
		if d.SecretRef == "" {
			return nil, fmt.Errorf("telegram destination needs --secret-ref secret:NAME")
		}
		token, err := resolve(d.SecretRef)
		if err != nil {
			return nil, err
		}
		if token == "" {
			return nil, fmt.Errorf("telegram bot token is empty")
		}
		me, err := telegramGetMe(ctx, token)
		if err != nil {
			return nil, err
		}
		return &Result{Dest: d.Name, Kind: d.Kind, Target: d.Target, Detail: "bot @" + me}, nil
	case store.DestMastodon:
		return testMastodon(ctx, d, resolve)
	case store.DestBluesky:
		return testBluesky(ctx, d, resolve)
	default:
		return nil, fmt.Errorf("destination kind %q is not supported", d.Kind)
	}
}

func publishFile(d *store.Dest, c Content) (*Result, error) {
	dir := strings.TrimSpace(d.Target)
	if dir == "" {
		return nil, fmt.Errorf("file destination needs --target DIR")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	stamp := time.Now().UTC().Format("20060102T150405Z")
	name := sanitizeFile(c.Name)
	if name == "" {
		name = "post"
	}
	base := filepath.Join(dir, stamp+"_"+name)
	var b strings.Builder
	if c.Title != "" {
		b.WriteString("# " + c.Title + "\n\n")
	}
	b.WriteString(c.Body)
	if !strings.HasSuffix(c.Body, "\n") {
		b.WriteByte('\n')
	}
	if len(c.Media) > 0 {
		b.WriteString("\n## Media\n")
		for _, m := range c.Media {
			b.WriteString("- " + m + "\n")
		}
	}
	path, err := writeNew(base, ".md", []byte(b.String()))
	if err != nil {
		return nil, err
	}
	return &Result{Dest: d.Name, Kind: d.Kind, Target: dir, Path: path, Detail: "wrote file"}, nil
}

func sanitizeFile(s string) string {
	s = strings.TrimSpace(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case r == ' ' || r == '.':
			b.WriteByte('_')
		}
	}
	return b.String()
}

// writeNew writes data to base+ext, or to base_2+ext, base_3+ext and so on
// when that name is taken: two posts in the same second never overwrite.
func writeNew(base, ext string, data []byte) (string, error) {
	for i := 1; i <= 1000; i++ {
		path := base + ext
		if i > 1 {
			path = fmt.Sprintf("%s_%d%s", base, i, ext)
		}
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		if _, err := f.Write(data); err != nil {
			f.Close()
			return "", err
		}
		return path, f.Close()
	}
	return "", fmt.Errorf("%s%s and 1000 numbered names after it already exist", base, ext)
}
