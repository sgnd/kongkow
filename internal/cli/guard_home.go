package cli

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sgnd/kongkow/internal/out"
)

// An AI agent's kongkow keeps the person's data folder, ~/.kongkow.
// KONGKOW_HOME moves it: pointed at a new folder, kongkow would know none
// of the person's protected destinations and would write its audit log
// there, out of the person's sight. So for an agent a KONGKOW_HOME that is
// another folder stops every command that reads data, a dry run too, and
// no flag lets it through. A person who keeps the data elsewhere can make
// ~/.kongkow a link to it: a link to the same folder counts as ~/.kongkow.
// The home folder comes from the system's user database, so an agent's
// HOME=/x does not move it.

// defaultHome is ~/.kongkow in the account's home folder. Tests replace it.
var defaultHome = func() string {
	if u, err := user.Current(); err == nil && u.HomeDir != "" {
		return filepath.Join(u.HomeDir, ".kongkow")
	}
	if d, err := os.UserHomeDir(); err == nil {
		return filepath.Join(d, ".kongkow")
	}
	return ".kongkow"
}

// sameDir reports whether a and b are one folder: the same path, or the
// same folder through a link.
func sameDir(a, b string) bool {
	if filepath.Clean(a) == filepath.Clean(b) {
		return true
	}
	fa, err := os.Stat(a)
	if err != nil {
		return false
	}
	fb, err := os.Stat(b)
	return err == nil && os.SameFile(fa, fb)
}

// homeFree are the commands that read no data, so they run anywhere.
var homeFree = map[string]bool{"kongkow": true, "help": true, "completion": true, "version": true, "guide": true}

// guardHome stops an AI agent's command when KONGKOW_HOME names another
// folder than ~/.kongkow. The stop is final: exit 6, no --yes.
func guardHome(cmd *cobra.Command) error {
	top := strings.Fields(cmd.CommandPath())
	name := "kongkow"
	if len(top) > 1 {
		name = top[1]
	}
	if homeFree[name] || !isAgent() {
		return nil
	}
	h := os.Getenv("KONGKOW_HOME")
	if h == "" || sameDir(h, defaultHome()) {
		return nil
	}
	return out.Errf(out.ExitPolicy, "OTHER_HOME",
		"take KONGKOW_HOME out of the command's environment and run it again; no flag lets an AI agent use another data folder",
		"KONGKOW_HOME points kongkow at %s, not at %s: its destinations, protection and audit log are not the usual ones, so an AI agent cannot use it",
		h, defaultHome())
}
