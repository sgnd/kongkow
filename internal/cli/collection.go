package cli

import (
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/sgnd/kongkow/internal/out"
	"github.com/sgnd/kongkow/internal/store"
)

func collectionCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "collection",
		Short: "Manage post collections",
	}
	c.AddCommand(collectionAddCmd(), collectionListCmd(), collectionShowCmd(), collectionRmCmd())
	return c
}

func collectionAddCmd() *cobra.Command {
	var title, notes, tags string
	c := &cobra.Command{
		Use:   "add NAME",
		Short: "Add a collection",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]
			now := time.Now().UTC()
			err := store.Update(func(s *store.State) error {
				if s.Collections[name] != nil {
					return out.Errf(out.ExitConflict, "EXISTS", "pick another name",
						"collection %q already exists", name)
				}
				s.Collections[name] = &store.Collection{
					Name: name, Title: title, Notes: notes, Tags: splitCSV(tags),
					CreatedAt: now, UpdatedAt: now,
				}
				return nil
			})
			if err != nil {
				return err
			}
			st, _ := store.Load()
			audit(st, "collection add", true, map[string]any{"name": name})
			out.P.Print("collection add", map[string]any{"name": name, "action": "created"}, nil)
			return nil
		},
	}
	c.Flags().StringVar(&title, "title", "", "title")
	c.Flags().StringVar(&notes, "notes", "", "notes")
	c.Flags().StringVar(&tags, "tags", "", "comma-separated tags")
	return c
}

type collectionListResult struct {
	Collections []collectionRow `json:"collections"`
}

type collectionRow struct {
	Name  string `json:"name"`
	Title string `json:"title,omitempty"`
	Posts int    `json:"posts"`
}

func (r collectionListResult) Text() string {
	if len(r.Collections) == 0 {
		return "No collections yet: kongkow collection add NAME\n"
	}
	var b strings.Builder
	for _, c := range r.Collections {
		line := c.Name
		if c.Title != "" {
			line += "  " + c.Title
		}
		line += "  " + countOf(c.Posts, "post")
		b.WriteString(line + "\n")
	}
	return b.String()
}

func collectionListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List collections",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			st, err := store.Load()
			if err != nil {
				return err
			}
			r := collectionListResult{Collections: []collectionRow{}}
			for _, c := range st.SortedCollections() {
				n := 0
				for _, p := range st.Posts {
					if p.Collection == c.Name {
						n++
					}
				}
				r.Collections = append(r.Collections, collectionRow{Name: c.Name, Title: c.Title, Posts: n})
			}
			out.P.Print("collection list", r, nil)
			return nil
		},
	}
}

func collectionShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show NAME",
		Short: "Show one collection",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			st, err := store.Load()
			if err != nil {
				return err
			}
			c := st.Collections[args[0]]
			if c == nil {
				return out.Errf(out.ExitNotFound, "COLLECTION_NOT_FOUND",
					"check the name; `kongkow collection list` shows collections",
					"collection %q is not in the store", args[0])
			}
			out.P.Print("collection show", c, nil)
			return nil
		},
	}
}

func collectionRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm NAME",
		Short: "Remove a collection (posts keep their bodies; collection field clears)",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]
			err := store.Update(func(s *store.State) error {
				if s.Collections[name] == nil {
					return out.Errf(out.ExitNotFound, "COLLECTION_NOT_FOUND",
						"check the name; `kongkow collection list` shows collections",
						"collection %q is not in the store", name)
				}
				delete(s.Collections, name)
				for _, p := range s.Posts {
					if p.Collection == name {
						p.Collection = ""
						p.UpdatedAt = time.Now().UTC()
					}
				}
				return nil
			})
			if err != nil {
				return err
			}
			st, _ := store.Load()
			audit(st, "collection rm", true, map[string]any{"name": name})
			out.P.Print("collection rm", map[string]any{"name": name, "action": "removed"}, nil)
			return nil
		},
	}
}
