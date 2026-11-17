package cli

import (
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/sgnd/kongkow/internal/out"
	"github.com/sgnd/kongkow/internal/store"
)

func bookmarkCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "bookmark",
		Short: "Manage bookmarks",
	}
	c.AddCommand(bookmarkAddCmd(), bookmarkListCmd(), bookmarkShowCmd(), bookmarkRmCmd())
	return c
}

func bookmarkAddCmd() *cobra.Command {
	var url, title, notes, tags string
	c := &cobra.Command{
		Use:   "add NAME",
		Short: "Add a bookmark",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if strings.TrimSpace(url) == "" {
				return out.Usage("need --url")
			}
			name := args[0]
			now := time.Now().UTC()
			err := store.Update(func(s *store.State) error {
				if s.Bookmarks[name] != nil {
					return out.Errf(out.ExitConflict, "EXISTS", "pick another name",
						"bookmark %q already exists", name)
				}
				s.Bookmarks[name] = &store.Bookmark{
					Name: name, URL: url, Title: title, Notes: notes, Tags: splitCSV(tags),
					CreatedAt: now, UpdatedAt: now,
				}
				return nil
			})
			if err != nil {
				return err
			}
			st, _ := store.Load()
			audit(st, "bookmark add", true, map[string]any{"name": name})
			out.P.Print("bookmark add", map[string]any{"name": name, "action": "created"}, nil)
			return nil
		},
	}
	c.Flags().StringVar(&url, "url", "", "URL")
	c.Flags().StringVar(&title, "title", "", "title")
	c.Flags().StringVar(&notes, "notes", "", "notes")
	c.Flags().StringVar(&tags, "tags", "", "comma-separated tags")
	return c
}

type bookmarkListResult struct {
	Bookmarks []bookmarkRow `json:"bookmarks"`
}

type bookmarkRow struct {
	Name  string `json:"name"`
	URL   string `json:"url"`
	Title string `json:"title,omitempty"`
}

func (r bookmarkListResult) Text() string {
	if len(r.Bookmarks) == 0 {
		return "No bookmarks yet: kongkow bookmark add NAME --url URL\n"
	}
	var b strings.Builder
	for _, bm := range r.Bookmarks {
		line := bm.Name + "  " + bm.URL
		if bm.Title != "" {
			line += "  " + bm.Title
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

func bookmarkListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List bookmarks",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			st, err := store.Load()
			if err != nil {
				return err
			}
			r := bookmarkListResult{Bookmarks: []bookmarkRow{}}
			for _, b := range st.SortedBookmarks() {
				r.Bookmarks = append(r.Bookmarks, bookmarkRow{Name: b.Name, URL: b.URL, Title: b.Title})
			}
			out.P.Print("bookmark list", r, nil)
			return nil
		},
	}
}

func bookmarkShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show NAME",
		Short: "Show one bookmark",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			st, err := store.Load()
			if err != nil {
				return err
			}
			b := st.Bookmarks[args[0]]
			if b == nil {
				return out.Errf(out.ExitNotFound, "BOOKMARK_NOT_FOUND",
					"check the name; `kongkow bookmark list` shows bookmarks",
					"bookmark %q is not in the store", args[0])
			}
			out.P.Print("bookmark show", b, nil)
			return nil
		},
	}
}

func bookmarkRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm NAME",
		Short: "Remove a bookmark",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]
			err := store.Update(func(s *store.State) error {
				if s.Bookmarks[name] == nil {
					return out.Errf(out.ExitNotFound, "BOOKMARK_NOT_FOUND",
						"check the name; `kongkow bookmark list` shows bookmarks",
						"bookmark %q is not in the store", name)
				}
				delete(s.Bookmarks, name)
				return nil
			})
			if err != nil {
				return err
			}
			st, _ := store.Load()
			audit(st, "bookmark rm", true, map[string]any{"name": name})
			out.P.Print("bookmark rm", map[string]any{"name": name, "action": "removed"}, nil)
			return nil
		},
	}
}
