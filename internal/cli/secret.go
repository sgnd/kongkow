package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/sgnd/kongkow/internal/out"
	"github.com/sgnd/kongkow/internal/store"
	"github.com/sgnd/kongkow/internal/vault"
)

func secretCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "secret",
		Short: "Manage the local vault",
	}
	c.AddCommand(secretSetCmd(), secretListCmd(), secretShowCmd(), secretRmCmd())
	return c
}

func secretSetCmd() *cobra.Command {
	var value string
	c := &cobra.Command{
		Use:   "set NAME",
		Short: "Store a secret in the vault",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]
			if value == "" {
				if !term.IsTerminal(int(os.Stdin.Fd())) {
					return out.Errf(out.ExitPolicy, "NEEDS_PERSON",
						"run `kongkow secret set NAME` at your terminal, or pass --value",
						"a secret value needs a person at a terminal or --value")
				}
				fmt.Fprint(os.Stderr, "Value: ")
				b, err := term.ReadPassword(int(os.Stdin.Fd()))
				fmt.Fprintln(os.Stderr)
				if err != nil {
					return err
				}
				value = string(b)
			}
			if err := vault.Set(name, value); err != nil {
				return err
			}
			st, _ := store.Load()
			audit(st, "secret set", true, map[string]any{"name": name})
			out.P.Print("secret set", map[string]any{"name": name, "action": "set"}, nil)
			return nil
		},
	}
	c.Flags().StringVar(&value, "value", "", "secret value (prefer prompting at a terminal)")
	return c
}

type secretListResult struct {
	Secrets []vault.Info `json:"secrets"`
}

func (r secretListResult) Text() string {
	if len(r.Secrets) == 0 {
		return "No secrets yet: kongkow secret set NAME\n"
	}
	var b strings.Builder
	for _, s := range r.Secrets {
		b.WriteString(s.Name + "\n")
	}
	return b.String()
}

func secretListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List secret names in the vault",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			list, err := vault.List()
			if err != nil {
				return err
			}
			out.P.Print("secret list", secretListResult{Secrets: list}, nil)
			return nil
		},
	}
}

func secretShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show NAME",
		Short: "Show a secret (person + --reveal)",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if !g.reveal {
				return out.Errf(out.ExitPolicy, "REVEAL_NEEDS_PERSON",
					"add --reveal at your own terminal when you want the value",
					"showing a secret needs --reveal").NoFlag()
			}
			if !term.IsTerminal(int(os.Stdin.Fd())) {
				return out.Errf(out.ExitPolicy, "REVEAL_NEEDS_PERSON",
					"run `kongkow secret show NAME --reveal` at your terminal",
					"showing a secret needs a person at a terminal").NoFlag()
			}
			v, err := vault.Get(args[0])
			if errors.Is(err, vault.ErrNotFound) {
				return out.Errf(out.ExitNotFound, "SECRET_NOT_FOUND",
					"check the name; `kongkow secret list` shows secrets",
					"secret %q is not in the vault", args[0])
			}
			if err != nil {
				return err
			}
			out.P.NoRedact = true
			out.P.Print("secret show", map[string]any{"name": args[0], "value": v}, nil)
			return nil
		},
	}
}

func secretRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm NAME",
		Short: "Remove a secret from the vault",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if err := vault.Delete(args[0]); err != nil {
				if errors.Is(err, vault.ErrNotFound) {
					return out.Errf(out.ExitNotFound, "SECRET_NOT_FOUND",
						"check the name; `kongkow secret list` shows secrets",
						"secret %q is not in the vault", args[0])
				}
				return err
			}
			st, _ := store.Load()
			audit(st, "secret rm", true, map[string]any{"name": args[0]})
			out.P.Print("secret rm", map[string]any{"name": args[0], "action": "removed"}, nil)
			return nil
		},
	}
}
