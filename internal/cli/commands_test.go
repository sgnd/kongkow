package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/sgnd/kongkow/internal/out"
	"github.com/sgnd/kongkow/internal/store"
	"github.com/sgnd/kongkow/internal/vault"
)

func freshCLI(t *testing.T) {
	t.Helper()
	t.Setenv("KONGKOW_HOME", t.TempDir())
	store.Close()
	t.Cleanup(store.Close)
}

// runJSON runs a command and returns its JSON envelope.
func runJSON(t *testing.T, want int, args ...string) map[string]any {
	t.Helper()
	resetOut()
	var buf bytes.Buffer
	out.P.W = &buf
	if code := executeArgs(append(args, "-o", "json")); code != want {
		t.Fatalf("%v: want exit %d, got %d: %s", args, want, code, buf.String())
	}
	env := map[string]any{}
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("%v: not JSON: %q", args, buf.String())
	}
	return env
}

func TestCollectionAndBookmark(t *testing.T) {
	freshCLI(t)
	runJSON(t, 0, "collection", "add", "c1", "--title", "News")
	runJSON(t, 0, "bookmark", "add", "b1", "--url", "https://example.com", "--title", "Ex")
	runJSON(t, 0, "collection", "show", "c1")
	runJSON(t, 0, "bookmark", "show", "b1")
	env := runJSON(t, 0, "list")
	s, _ := json.Marshal(env["data"])
	if !strings.Contains(string(s), "c1") || !strings.Contains(string(s), "b1") {
		t.Fatalf("list misses items: %s", s)
	}
	runJSON(t, 0, "collection", "rm", "c1")
	runJSON(t, 0, "bookmark", "rm", "b1")
	runJSON(t, out.ExitNotFound, "collection", "show", "c1")
	runJSON(t, out.ExitNotFound, "bookmark", "show", "b1")
}

func TestSecretNeverShownWithoutReveal(t *testing.T) {
	freshCLI(t)
	if err := vault.Set("tg", "VERY-SECRET-VALUE"); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"secret", "list"}, {"list"}, {"doctor"}} {
		resetOut()
		var buf bytes.Buffer
		out.P.W = &buf
		executeArgs(append(args, "-o", "json"))
		if strings.Contains(buf.String(), "VERY-SECRET-VALUE") {
			t.Fatalf("%v leaked the secret", args)
		}
	}
	runJSON(t, out.ExitPolicy, "secret", "show", "tg")
}

func TestAuditRecordsMutations(t *testing.T) {
	freshCLI(t)
	runJSON(t, 0, "post", "draft", "n1", "--body", "hi")
	env := runJSON(t, 0, "audit", "--limit", "10")
	s, _ := json.Marshal(env["data"])
	if !strings.Contains(string(s), "post draft") {
		t.Fatalf("audit misses post draft: %s", s)
	}
}

func TestDoctorOnFreshHome(t *testing.T) {
	freshCLI(t)
	env := runJSON(t, 0, "doctor")
	if env["ok"] != true {
		t.Fatalf("doctor: %v", env)
	}
}
