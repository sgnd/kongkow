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

func TestPublishFileNeverOverwrites(t *testing.T) {
	dir := t.TempDir()
	d := &store.Dest{Name: "out", Kind: store.DestFile, Target: dir}
	// Three publishes in the same second used to land on one file name.
	seen := map[string]bool{}
	for i := 0; i < 3; i++ {
		r, err := Publish(t.Context(), d, Content{Name: "hello", Body: "world"}, nil)
		if err != nil {
			t.Fatal(err)
		}
		seen[r.Path] = true
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) != 3 || len(seen) != 3 {
		t.Fatalf("want 3 files, got %d on disk, %d paths", len(ents), len(seen))
	}
}

func TestWriteNewNumbersTakenNames(t *testing.T) {
	base := filepath.Join(t.TempDir(), "20261118T112616Z_hello")
	p1, err1 := writeNew(base, ".md", []byte("one"))
	p2, err2 := writeNew(base, ".md", []byte("two"))
	if err1 != nil || err2 != nil {
		t.Fatal(err1, err2)
	}
	if p1 != base+".md" || p2 != base+"_2.md" {
		t.Fatalf("paths %q %q", p1, p2)
	}
	if b, _ := os.ReadFile(p1); string(b) != "one" {
		t.Fatalf("first file was overwritten: %q", b)
	}
}
