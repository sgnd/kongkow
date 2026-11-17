package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sgnd/kongkow/internal/out"
	"github.com/sgnd/kongkow/internal/store"
)

func resetOut() {
	out.P = &out.Printer{Format: "json", W: io.Discard, Version: "test"}
	g = globalFlags{}
	store.DryRun = false
}

func TestSmokeFilePublish(t *testing.T) {
	home := t.TempDir()
	t.Setenv("KONGKOW_HOME", home)
	store.Close()
	resetOut()
	outDir := filepath.Join(home, "out")
	if err := os.MkdirAll(outDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if code := executeArgs([]string{"post", "draft", "n1", "--body", "hi", "-o", "json"}); code != 0 {
		t.Fatalf("draft exit %d", code)
	}
	resetOut()
	if code := executeArgs([]string{"dest", "add", "f1", "--kind", "file", "--target", outDir, "-o", "json"}); code != 0 {
		t.Fatalf("dest add exit %d", code)
	}
	resetOut()
	if code := executeArgs([]string{"post", "publish", "n1", "--to", "f1", "-o", "json"}); code != 0 {
		t.Fatalf("publish exit %d", code)
	}
	ents, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) != 1 || !strings.HasSuffix(ents[0].Name(), ".md") {
		t.Fatalf("files %#v", ents)
	}
}

func TestProtectedNeedsYes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("KONGKOW_HOME", home)
	store.Close()
	resetOut()
	outDir := filepath.Join(home, "out")
	_ = os.MkdirAll(outDir, 0o700)
	if code := executeArgs([]string{"post", "draft", "n1", "--body", "hi", "-o", "json"}); code != 0 {
		t.Fatal(code)
	}
	resetOut()
	if code := executeArgs([]string{"dest", "add", "p1", "--kind", "file", "--target", outDir, "--protected", "-o", "json"}); code != 0 {
		t.Fatal(code)
	}
	resetOut()
	code := executeArgs([]string{"post", "publish", "n1", "--to", "p1", "-o", "json"})
	if code != out.ExitPolicy {
		t.Fatalf("want exit %d got %d", out.ExitPolicy, code)
	}
	resetOut()
	if code := executeArgs([]string{"post", "publish", "n1", "--to", "p1", "--yes", "-o", "json"}); code != 0 {
		t.Fatalf("with --yes: %d", code)
	}
}
