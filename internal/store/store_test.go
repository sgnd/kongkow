package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHomeEnv(t *testing.T) {
	t.Setenv("KONGKOW_HOME", "/tmp/kongkow-test-home")
	if Home() != "/tmp/kongkow-test-home" {
		t.Fatalf("Home() = %q", Home())
	}
}

func TestEnsureHome(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Join(dir, ".kongkow")
	t.Setenv("KONGKOW_HOME", home)
	Close()
	if err := EnsureHome(); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(home)
	if err != nil {
		t.Fatal(err)
	}
	if !fi.IsDir() {
		t.Fatal("not a directory")
	}
}
