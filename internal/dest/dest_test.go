package dest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sgnd/kongkow/internal/store"
)

func TestPublishFile(t *testing.T) {
	dir := t.TempDir()
	d := &store.Dest{Name: "out", Kind: store.DestFile, Target: dir}
	r, err := Publish(t.Context(), d, Content{Name: "hello", Title: "Hi", Body: "world"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r.Path == "" {
		t.Fatal("empty path")
	}
	b, err := os.ReadFile(r.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(r.Path) {
		t.Fatalf("path %q", r.Path)
	}
	if string(b) == "" || !contains(string(b), "world") {
		t.Fatalf("body %q", b)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(len(s) > 0 && (func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		})()))
}
