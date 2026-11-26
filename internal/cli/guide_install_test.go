package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sgnd/kongkow/skills"
)

func skillPaths(base string) []string {
	return []string{
		filepath.Join(base, ".claude", "skills", "kongkow", "SKILL.md"),
		filepath.Join(base, ".agents", "skills", "kongkow", "SKILL.md"),
	}
}

func installActions(t *testing.T, env map[string]any) string {
	t.Helper()
	data, _ := env["data"].(map[string]any)
	files, _ := data["files"].([]any)
	var got []string
	for _, f := range files {
		got = append(got, f.(map[string]any)["action"].(string))
	}
	return strings.Join(got, ",")
}

func fakeHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}

func TestGuideInstallHome(t *testing.T) {
	freshCLI(t)
	home := fakeHome(t)
	env := runJSON(t, 0, "guide", "install")
	if got := installActions(t, env); got != "created,created" {
		t.Fatalf("actions = %s", got)
	}
	for _, p := range skillPaths(home) {
		if b, err := os.ReadFile(p); err != nil || string(b) != skills.Kongkow {
			t.Errorf("%s: err %v, content differs from the embedded skill", p, err)
		}
	}
	if got := installActions(t, runJSON(t, 0, "guide", "install")); got != "unchanged,unchanged" {
		t.Errorf("second run actions = %s", got)
	}
}

func TestGuideInstallKeepsOldCopy(t *testing.T) {
	freshCLI(t)
	home := fakeHome(t)
	p := skillPaths(home)[0]
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("my own notes"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := installActions(t, runJSON(t, 0, "guide", "install")); got != "updated,created" {
		t.Fatalf("actions = %s", got)
	}
	if b, err := os.ReadFile(p + ".bak"); err != nil || string(b) != "my own notes" {
		t.Errorf("backup: %v %q", err, b)
	}
}

func TestGuideInstallDryRunWritesNothing(t *testing.T) {
	freshCLI(t)
	home := fakeHome(t)
	env := runJSON(t, 0, "guide", "install", "--dry-run")
	if got := installActions(t, env); got != "created,created" {
		t.Errorf("plan = %s", got)
	}
	if entries, _ := os.ReadDir(home); len(entries) != 0 {
		t.Errorf("dry run wrote %d entries", len(entries))
	}
}

func TestGuideInstallProject(t *testing.T) {
	freshCLI(t)
	home, proj := fakeHome(t), t.TempDir()
	t.Chdir(proj)
	runJSON(t, 0, "guide", "install", "--project")
	for _, p := range skillPaths(proj) {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("project install missing %s", p)
		}
	}
	if entries, _ := os.ReadDir(home); len(entries) != 0 {
		t.Error("--project wrote into the home folder")
	}
}

func TestGuideInstallProjectRefusesLink(t *testing.T) {
	freshCLI(t)
	fakeHome(t)
	proj, elsewhere := t.TempDir(), t.TempDir()
	if err := os.Symlink(elsewhere, filepath.Join(proj, ".claude")); err != nil {
		t.Skip("no symlinks here")
	}
	t.Chdir(proj)
	env := runJSON(t, 1, "guide", "install", "--project")
	if e, _ := env["error"].(map[string]any); e["code"] != "WRITE_FAILED" {
		t.Fatalf("error %v", e)
	}
	if entries, _ := os.ReadDir(elsewhere); len(entries) != 0 {
		t.Error("wrote through the link")
	}
}

func TestGuideStillPrintsPlaybookAndTopics(t *testing.T) {
	freshCLI(t)
	data, _ := runJSON(t, 0, "guide")["data"].(map[string]any)
	if !strings.Contains(data["text"].(string), "kongkow guide install") {
		t.Error("the playbook must point to guide install")
	}
}
