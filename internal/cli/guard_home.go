package cli

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sgnd/kongkow/internal/out"
	"github.com/sgnd/kongkow/internal/store"
)

// An AI agent's kongkow works only in the person's ~/.kongkow, found through
// the user database, so neither KONGKOW_HOME nor HOME can move it elsewhere.
// A link to ~/.kongkow counts as ~/.kongkow.

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

// guardHome stops an AI agent's command when its data folder is not
// ~/.kongkow. The stop is final: exit 6, no --yes.
func guardHome(cmd *cobra.Command) error {
	top := strings.Fields(cmd.CommandPath())
	name := "kongkow"
	if len(top) > 1 {
		name = top[1]
	}
	if homeFree[name] || !isAgent() {
		return nil
	}
	h := store.Home()
	if sameDir(h, defaultHome()) {
		return nil
	}
	return out.Errf(out.ExitPolicy, "OTHER_HOME",
		"take KONGKOW_HOME and any changed HOME out of the command's environment and run it again; no flag lets an AI agent use another data folder",
		"kongkow would use %s, not %s: its destinations, protection and audit log are not the usual ones, so an AI agent cannot use it",
		h, defaultHome())
}
