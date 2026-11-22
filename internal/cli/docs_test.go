package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"
)

// metaCommands are not tools: the docs lists leave them out.
var metaCommands = map[string]bool{"version": true, "list": true, "completion": true, "help": true}

// commandSet is every command the binary has: "noun verb" for a command with
// subcommands, "noun" for one without (audit, doctor).
func commandSet() []string {
	var got []string
	for _, n := range newRoot().Commands() {
		if n.Hidden || metaCommands[n.Name()] {
			continue
		}
		subs := 0
		for _, v := range n.Commands() {
			if !v.Hidden && v.Name() != "help" {
				got = append(got, n.Name()+" "+v.Name())
				subs++
			}
		}
		if subs == 0 {
			got = append(got, n.Name())
		}
	}
	sort.Strings(got)
	return got
}

func repoFile(t *testing.T, parts ...string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(append([]string{"..", ".."}, parts...)...))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func sameSet(t *testing.T, where string, got, want []string) {
	t.Helper()
	sort.Strings(got)
	got = slices.Compact(got)
	for _, w := range want {
		if !slices.Contains(got, w) {
			t.Errorf("%s does not list `kongkow %s`, but the binary has it", where, w)
		}
	}
	for _, g := range got {
		if !slices.Contains(want, g) {
			t.Errorf("%s lists `kongkow %s`, but the binary has no such command", where, g)
		}
	}
}

func commandsSection(t *testing.T, body string) string {
	t.Helper()
	const head = "\n## Commands\n"
	i := strings.Index(body, head)
	if i < 0 {
		t.Fatal("no \"## Commands\" section")
	}
	rest := body[i+len(head):]
	if j := strings.Index(rest, "\n## "); j >= 0 {
		rest = rest[:j]
	}
	return rest
}

var commandLine = regexp.MustCompile("(?m)^- `kongkow ([a-z-]+(?: [a-z-]+)?)`")

func listedCommands(t *testing.T, file, body string) []string {
	t.Helper()
	var got []string
	for _, m := range commandLine.FindAllStringSubmatch(commandsSection(t, body), -1) {
		got = append(got, m[1])
	}
	if len(got) == 0 {
		t.Fatalf("%s: no command lines in its Commands section", file)
	}
	return got
}

func TestSkillListsEveryCommand(t *testing.T) {
	sameSet(t, "skills/kongkow/SKILL.md", listedCommands(t, "SKILL.md", repoFile(t, "skills", "kongkow", "SKILL.md")), commandSet())
}

func TestAgentsListsEveryCommand(t *testing.T) {
	sameSet(t, "AGENTS.md", listedCommands(t, "AGENTS.md", repoFile(t, "AGENTS.md")), commandSet())
}

// TestReadmeListsEveryCommand reads the command table: a first cell such as
// `post draft / compose` stands for `post draft` and `post compose`.
func TestReadmeListsEveryCommand(t *testing.T) {
	var got []string
	for _, line := range strings.Split(commandsSection(t, repoFile(t, "README.md")), "\n") {
		cells := strings.Split(line, "|")
		if len(cells) < 3 || !strings.HasPrefix(strings.TrimSpace(cells[1]), "`") {
			continue
		}
		parts := strings.Fields(strings.ReplaceAll(cells[1], "`", ""))
		if len(parts) == 0 || metaCommands[parts[0]] {
			continue
		}
		if len(parts) == 1 {
			got = append(got, parts[0])
			continue
		}
		for _, v := range strings.Split(strings.Join(parts[1:], " "), " / ") {
			got = append(got, parts[0]+" "+strings.TrimSpace(v))
		}
	}
	sameSet(t, "README.md", got, commandSet())
}

func TestEveryCommandHasHelp(t *testing.T) {
	for _, n := range newRoot().Commands() {
		if n.Hidden || n.Name() == "help" || n.Name() == "completion" {
			continue
		}
		if n.Short == "" {
			t.Errorf("%s has no short help", n.CommandPath())
		}
		for _, v := range n.Commands() {
			if !v.Hidden && v.Name() != "help" && v.Short == "" {
				t.Errorf("%s has no short help", v.CommandPath())
			}
		}
	}
}

// The release workflow cuts the notes from the line "## vX.Y.Z " (with the
// space), so every release heading must be exactly "## vX.Y.Z - YYYY-MM-DD".
func TestChangelogHeadingsMatchTheReleaseWorkflow(t *testing.T) {
	heading := regexp.MustCompile(`^## v\d+\.\d+\.\d+ - \d{4}-\d{2}-\d{2}$`)
	n := 0
	for _, line := range strings.Split(repoFile(t, "CHANGELOG.md"), "\n") {
		if strings.HasPrefix(line, "## ") {
			n++
			if !heading.MatchString(line) {
				t.Errorf("CHANGELOG heading %q is not \"## vX.Y.Z - YYYY-MM-DD\"", line)
			}
		}
	}
	if n == 0 {
		t.Fatal("no release headings in CHANGELOG.md")
	}
}
