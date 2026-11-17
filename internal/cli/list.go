package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sgnd/kongkow/internal/out"
	"github.com/sgnd/kongkow/internal/store"
)

type listItem struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
	Info string `json:"info,omitempty"`
}

type listResult struct {
	Items []listItem `json:"items"`
}

func (r listResult) Text() string {
	if len(r.Items) == 0 {
		return "Nothing added yet: kongkow post draft NAME  or  kongkow dest add NAME\n"
	}
	var b strings.Builder
	for _, it := range r.Items {
		line := it.Kind + "  " + it.Name
		if it.Info != "" {
			line += "  " + it.Info
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List posts, destinations, collections and bookmarks",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			st, err := store.Load()
			if err != nil {
				return err
			}
			r := listResult{Items: []listItem{}}
			for _, p := range st.SortedPosts() {
				info := p.Title
				if info == "" {
					info = fmt.Sprintf("%d bytes", len(p.Body))
				}
				r.Items = append(r.Items, listItem{Kind: "post", Name: p.Name, Info: info})
			}
			for _, d := range st.SortedDests() {
				info := d.Kind
				if d.Protected {
					info += " protected"
				}
				r.Items = append(r.Items, listItem{Kind: "dest", Name: d.Name, Info: info})
			}
			for _, c := range st.SortedCollections() {
				r.Items = append(r.Items, listItem{Kind: "collection", Name: c.Name, Info: c.Title})
			}
			for _, b := range st.SortedBookmarks() {
				r.Items = append(r.Items, listItem{Kind: "bookmark", Name: b.Name, Info: b.URL})
			}
			out.P.Print("list", r, nil)
			return nil
		},
	}
}
