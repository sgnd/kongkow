package cli

import (
	"context"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/sgnd/kongkow/internal/dest"
	"github.com/sgnd/kongkow/internal/out"
	"github.com/sgnd/kongkow/internal/store"
	"github.com/sgnd/kongkow/internal/vault"
)

func destCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "dest",
		Short: "Manage publish destinations",
	}
	c.AddCommand(destAddCmd(), destEditCmd(), destRmCmd(), destListCmd(), destShowCmd(), destTestCmd())
	return c
}

func destAddCmd() *cobra.Command {
	var kind, target, secretRef, notes string
	var protected bool
	c := &cobra.Command{
		Use:   "add NAME",
		Short: "Add a publish destination",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]
			if err := validateDestKind(kind, target, secretRef); err != nil {
				return err
			}
			now := time.Now().UTC()
			err := store.Update(func(s *store.State) error {
				if s.Dests[name] != nil {
					return out.Errf(out.ExitConflict, "EXISTS", "use `kongkow dest edit "+name+"` or pick another name",
						"destination %q already exists", name)
				}
				s.Dests[name] = &store.Dest{
					Name: name, Kind: kind, Target: target, SecretRef: secretRef,
					Protected: protected, Notes: notes, CreatedAt: now, UpdatedAt: now,
				}
				return nil
			})
			if err != nil {
				return err
			}
			st, _ := store.Load()
			audit(st, "dest add", true, map[string]any{"name": name, "kind": kind})
			out.P.Print("dest add", map[string]any{"name": name, "kind": kind, "action": "created"}, nil)
			return nil
		},
	}
	c.Flags().StringVar(&kind, "kind", "", strings.Join(store.DestKinds, "|"))
	c.Flags().StringVar(&target, "target", "", "chat id (telegram), instance URL (mastodon), handle (bluesky) or directory (file)")
	c.Flags().StringVar(&secretRef, "secret-ref", "", "secret:NAME for the token: bot token (telegram), access token (mastodon), app password (bluesky)")
	c.Flags().StringVar(&notes, "notes", "", "notes")
	c.Flags().BoolVar(&protected, "protected", false, "changes and publishes need --yes")
	return c
}

func destEditCmd() *cobra.Command {
	var target, secretRef, notes string
	var protected bool
	c := &cobra.Command{
		Use:   "edit NAME",
		Short: "Edit a publish destination",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			err := store.Update(func(s *store.State) error {
				d := s.Dests[name]
				if d == nil {
					return out.Errf(out.ExitNotFound, "DEST_NOT_FOUND",
						"check the name; `kongkow dest list` shows destinations",
						"destination %q is not in the store", name)
				}
				if err := protectDest(d, "editing"); err != nil {
					return err
				}
				if cmd.Flags().Changed("target") {
					d.Target = target
				}
				if cmd.Flags().Changed("secret-ref") {
					d.SecretRef = secretRef
				}
				if cmd.Flags().Changed("notes") {
					d.Notes = notes
				}
				if cmd.Flags().Changed("protected") {
					d.Protected = protected
				}
				if err := validateDestKind(d.Kind, d.Target, d.SecretRef); err != nil {
					return err
				}
				d.UpdatedAt = time.Now().UTC()
				return nil
			})
			if err != nil {
				return err
			}
			st, _ := store.Load()
			audit(st, "dest edit", true, map[string]any{"name": name})
			out.P.Print("dest edit", map[string]any{"name": name, "action": "changed"}, nil)
			return nil
		},
	}
	c.Flags().StringVar(&target, "target", "", "chat id or directory")
	c.Flags().StringVar(&secretRef, "secret-ref", "", "secret:NAME")
	c.Flags().StringVar(&notes, "notes", "", "notes")
	c.Flags().BoolVar(&protected, "protected", false, "require --yes for changes and publishes")
	return c
}

func destRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm NAME",
		Short: "Remove a publish destination",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]
			err := store.Update(func(s *store.State) error {
				d := s.Dests[name]
				if d == nil {
					return out.Errf(out.ExitNotFound, "DEST_NOT_FOUND",
						"check the name; `kongkow dest list` shows destinations",
						"destination %q is not in the store", name)
				}
				if err := protectDest(d, "removing"); err != nil {
					return err
				}
				delete(s.Dests, name)
				return nil
			})
			if err != nil {
				return err
			}
			st, _ := store.Load()
			audit(st, "dest rm", true, map[string]any{"name": name})
			out.P.Print("dest rm", map[string]any{"name": name, "action": "removed"}, nil)
			return nil
		},
	}
}

type destListResult struct {
	Dests []destRow `json:"dests"`
	Kinds []string  `json:"kinds"`
}

type destRow struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Target    string `json:"target,omitempty"`
	Protected bool   `json:"protected,omitempty"`
}

func (r destListResult) Text() string {
	var b strings.Builder
	if len(r.Dests) == 0 {
		b.WriteString("No destinations yet. Kinds: " + strings.Join(r.Kinds, ", ") + "\n")
		b.WriteString("Add: kongkow dest add NAME --kind file --target DIR\n")
		b.WriteString("     kongkow dest add NAME --kind telegram --target CHAT_ID --secret-ref secret:TOKEN\n")
		return b.String()
	}
	for _, d := range r.Dests {
		line := d.Name + "  " + d.Kind
		if d.Target != "" {
			line += "  " + d.Target
		}
		if d.Protected {
			line += "  protected"
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

func destListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List publish destinations",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			st, err := store.Load()
			if err != nil {
				return err
			}
			r := destListResult{Kinds: append([]string{}, store.DestKinds...), Dests: []destRow{}}
			for _, d := range st.SortedDests() {
				r.Dests = append(r.Dests, destRow{Name: d.Name, Kind: d.Kind, Target: d.Target, Protected: d.Protected})
			}
			out.P.Print("dest list", r, nil)
			return nil
		},
	}
}

func destShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show NAME",
		Short: "Show one destination (secret refs, not values)",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			st, err := store.Load()
			if err != nil {
				return err
			}
			d := st.Dests[args[0]]
			if d == nil {
				return out.Errf(out.ExitNotFound, "DEST_NOT_FOUND",
					"check the name; `kongkow dest list` shows destinations",
					"destination %q is not in the store", args[0])
			}
			out.P.Print("dest show", d, nil)
			return nil
		},
	}
}

func destTestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test NAME",
		Short: "Check that a destination is usable",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			st, err := store.Load()
			if err != nil {
				return err
			}
			d := st.Dests[args[0]]
			if d == nil {
				return out.Errf(out.ExitNotFound, "DEST_NOT_FOUND",
					"check the name; `kongkow dest list` shows destinations",
					"destination %q is not in the store", args[0])
			}
			if g.dryRun {
				out.P.Print("dest test", map[string]any{"name": d.Name, "kind": d.Kind, "action": "would_test"}, nil)
				return nil
			}
			res, err := dest.Test(context.Background(), d, vault.Resolve)
			if err != nil {
				return err
			}
			out.P.Print("dest test", map[string]any{
				"name": d.Name, "kind": d.Kind, "target": res.Target, "detail": res.Detail, "ok": true,
			}, nil)
			return nil
		},
	}
}

func validateDestKind(kind, target, secretRef string) error {
	switch kind {
	case store.DestFile:
		if strings.TrimSpace(target) == "" {
			return out.Usage("file destinations need --target DIR")
		}
	case store.DestTelegram, store.DestMastodon, store.DestBluesky:
		if strings.TrimSpace(target) == "" {
			return out.Usage("%s destinations need --target %s", kind, destTargetHelp[kind])
		}
		if kind == store.DestMastodon {
			if _, err := dest.MastodonBase(target); err != nil {
				return out.Usage("%v", err)
			}
		}
		if secretRef == "" {
			return out.Usage("%s destinations need --secret-ref secret:NAME", kind)
		}
		if !vault.ValidRef(secretRef) || !strings.HasPrefix(secretRef, "secret:") {
			return out.Usage("--secret-ref must be secret:NAME (never put the token on the command line)")
		}
	case "":
		return out.Usage("--kind is one of %s", strings.Join(store.DestKinds, ", "))
	default:
		return out.Usage("kind %q is not supported; use one of %s", kind, strings.Join(store.DestKinds, ", "))
	}
	return nil
}

var destTargetHelp = map[string]string{
	store.DestTelegram: "CHAT_ID",
	store.DestMastodon: "https://INSTANCE",
	store.DestBluesky:  "HANDLE",
}
