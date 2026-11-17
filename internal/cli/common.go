package cli

import (
	"os"
	"strings"
	"time"

	"github.com/sgnd/kongkow/internal/out"
	"github.com/sgnd/kongkow/internal/store"
)

func loadState() (*store.State, error) {
	return store.Load()
}

func requireYes(code, msg, hint string) error {
	if g.yes {
		return nil
	}
	return out.Errf(out.ExitPolicy, code, hint, "%s", msg)
}

func protectDest(d *store.Dest, action string) error {
	if d == nil || !d.Protected {
		return nil
	}
	return requireYes("PROTECTED_TARGET",
		"destination "+d.Name+" is protected; "+action+" needs a person's yes",
		"add --yes after the person agrees, or run the command at their terminal")
}

func parseTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04",
		"2006-01-02",
	}
	var last error
	for _, layout := range layouts {
		t, err := time.ParseInLocation(layout, s, time.Local)
		if err == nil {
			return t, nil
		}
		last = err
	}
	return time.Time{}, out.Usage("time %q is not understood (use RFC3339 or 2006-01-02 15:04): %v", s, last)
}

func actor() string {
	if a := os.Getenv("KONGKOW_ACTOR"); a != "" {
		return a
	}
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	return "unknown"
}

func isAgent() bool {
	if os.Getenv("KONGKOW_AGENT") == "1" {
		return true
	}
	for _, k := range []string{"CLAUDECODE", "CODEX_THREAD_ID", "CURSOR_AGENT", "GEMINI_CLI"} {
		if os.Getenv(k) != "" {
			return true
		}
	}
	return false
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func readBodyFlag(body, bodyFile string) (string, error) {
	if bodyFile != "" {
		b, err := os.ReadFile(bodyFile)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	return body, nil
}
