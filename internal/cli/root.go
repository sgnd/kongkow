// Package cli wires every kongkow command.
package cli

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/sgnd/kongkow/internal/out"
	"github.com/sgnd/kongkow/internal/store"
	"github.com/sgnd/kongkow/internal/vault"
)

var Version = "dev"

type globalFlags struct {
	output string
	pretty bool
	dryRun bool
	yes    bool
	reveal bool
}

var g globalFlags

type exitCode int

func (e exitCode) Error() string { return "exit " + strconv.Itoa(int(e)) }

func Execute() int {
	vault.TakeUnlockEnv()
	return executeArgs(os.Args[1:])
}

func executeArgs(args []string) int {
	defer store.Close()
	if Version == "dev" {
		if bi, ok := debug.ReadBuildInfo(); ok && isRelease(bi.Main.Version) {
			Version = bi.Main.Version
		}
	}
	out.P.Version = Version
	if f, ok := outputFormat(os.Getenv("KONGKOW_OUTPUT")); ok {
		out.P.Format = f
	}
	root := newRoot()
	root.SetArgs(args)
	cmd, err := root.ExecuteC()
	if err == nil {
		return 0
	}
	var ec exitCode
	if errors.As(err, &ec) {
		return int(ec)
	}
	name := "kongkow"
	if cmd != nil {
		name = envelopeName(cmd)
	}
	return out.P.Fail(name, err)
}

func isRelease(v string) bool {
	return v != "" && v != "(devel)" && !strings.HasPrefix(v, "v0.0.0-")
}

func outputFormat(s string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "json":
		return "json", true
	case "text":
		return "text", true
	}
	return "", false
}

func envelopeName(cmd *cobra.Command) string {
	parts := strings.Fields(cmd.CommandPath())
	if len(parts) <= 1 {
		return "kongkow"
	}
	return strings.Join(parts[1:], " ")
}

func newRoot() *cobra.Command {
	root := &cobra.Command{
		Use:           "kongkow",
		Short:         "Local-first drafts and publish destinations, for people and AI agents",
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			out.P.Reset()
			out.P.Start = time.Now()
			out.P.Version = Version
			out.P.DryRun = g.dryRun
			store.DryRun = g.dryRun
			switch {
			case g.output != "":
				out.P.Format = g.output
			default:
				if f, ok := outputFormat(os.Getenv("KONGKOW_OUTPUT")); ok {
					out.P.Format = f
				} else if term.IsTerminal(int(os.Stdout.Fd())) {
					out.P.Format = "text"
				} else {
					out.P.Format = "json"
				}
			}
			out.P.Pretty = g.pretty
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	root.PersistentFlags().StringVarP(&g.output, "output", "o", "", "json|text (default: text at a terminal, json when piped)")
	root.PersistentFlags().BoolVar(&g.pretty, "pretty", false, "indent JSON")
	root.PersistentFlags().BoolVar(&g.dryRun, "dry-run", false, "show what would change without writing")
	root.PersistentFlags().BoolVar(&g.yes, "yes", false, "confirm a change that needs a person's yes")
	root.PersistentFlags().BoolVar(&g.reveal, "reveal", false, "show a secret value (person at a terminal only)")

	root.AddCommand(
		versionCmd(),
		doctorCmd(),
		listCmd(),
		guideCmd(),
		secretCmd(),
		destCmd(),
		postCmd(),
		collectionCmd(),
		bookmarkCmd(),
		auditCmd(),
	)
	return root
}

func toErr(err error) *out.Error {
	var e *out.Error
	if errors.As(err, &e) {
		return e
	}
	return out.Errf(out.ExitError, "ERROR", "read the message; `kongkow doctor` checks this computer's setup", "%v", err)
}

func countOf(n int, singular string) string {
	if n == 1 {
		return "1 " + singular
	}
	return fmt.Sprintf("%d %ss", n, singular)
}
