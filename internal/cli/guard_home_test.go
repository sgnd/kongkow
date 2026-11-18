package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sgnd/kongkow/internal/out"
	"github.com/sgnd/kongkow/internal/store"
)

func asAgent(t *testing.T, usual string) {
	t.Helper()
	oldVars, oldHome := agentVars, defaultHome
	agentVars = []string{"KONGKOW_TEST_AGENT"}
	defaultHome = func() string { return usual }
	t.Setenv("KONGKOW_TEST_AGENT", "")
	t.Cleanup(func() { agentVars, defaultHome = oldVars, oldHome })
}

func TestAgentStoppedInOtherHome(t *testing.T) {
	usual := filepath.Join(t.TempDir(), ".kongkow")
	asAgent(t, usual)
	t.Setenv("KONGKOW_HOME", filepath.Join(t.TempDir(), "elsewhere"))
	store.Close()
	t.Cleanup(store.Close)
	run(t, out.ExitPolicy, "list")
	run(t, out.ExitPolicy, "post", "draft", "x", "--body", "y", "--dry-run")
	run(t, out.ExitPolicy, "post", "draft", "x", "--body", "y", "--yes")
	run(t, 0, "version")
	run(t, 0, "guide")
}

func TestAgentStoppedWhenHomeMoves(t *testing.T) {
	usual := filepath.Join(t.TempDir(), ".kongkow")
	asAgent(t, usual)
	t.Setenv("KONGKOW_HOME", "")
	t.Setenv("HOME", t.TempDir())
	store.Close()
	t.Cleanup(store.Close)
	run(t, out.ExitPolicy, "list")
}

func TestAgentAllowedInUsualHome(t *testing.T) {
	usual := filepath.Join(t.TempDir(), ".kongkow")
	asAgent(t, usual)
	store.Close()
	t.Cleanup(store.Close)

	t.Setenv("KONGKOW_HOME", usual)
	run(t, 0, "list")

	// A link to the usual folder counts as the usual folder.
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(usual, link); err != nil {
		t.Skip("no symlinks here:", err)
	}
	store.Close()
	t.Setenv("KONGKOW_HOME", link)
	run(t, 0, "list")
}

func TestPersonNotStopped(t *testing.T) {
	t.Setenv("KONGKOW_HOME", t.TempDir())
	store.Close()
	t.Cleanup(store.Close)
	run(t, 0, "list") // agentVars is empty in tests: a person
}
