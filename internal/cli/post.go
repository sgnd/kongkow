package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/sgnd/kongkow/internal/dest"
	"github.com/sgnd/kongkow/internal/out"
	"github.com/sgnd/kongkow/internal/store"
	"github.com/sgnd/kongkow/internal/vault"
)

func postCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "post",
		Short: "Manage local posts and publish them",
	}
	c.AddCommand(
		postDraftCmd(),
		postComposeCmd(),
		postListCmd(),
		postShowCmd(),
		postEditCmd(),
		postRmCmd(),
		postPublishCmd(),
		postScheduleCmd(),
		postStatusCmd(),
		postCancelCmd(),
	)
	return c
}

func postDraftCmd() *cobra.Command {
	var title, body, bodyFile, collection, media, tags string
	c := &cobra.Command{
		Use:   "draft NAME",
		Short: "Create a local post draft",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]
			text, err := readBodyFlag(body, bodyFile)
			if err != nil {
				return err
			}
			now := time.Now().UTC()
			err = store.Update(func(s *store.State) error {
				if s.Posts[name] != nil {
					return out.Errf(out.ExitConflict, "EXISTS", "use `kongkow post edit "+name+"` or pick another name",
						"post %q already exists", name)
				}
				if collection != "" && s.Collections[collection] == nil {
					return out.Errf(out.ExitNotFound, "COLLECTION_NOT_FOUND",
						"check the name; `kongkow collection list` shows collections",
						"collection %q is not in the store", collection)
				}
				s.Posts[name] = &store.Post{
					Name: name, Title: title, Body: text, Collection: collection,
					Media: splitCSV(media), Tags: splitCSV(tags),
					CreatedAt: now, UpdatedAt: now,
				}
				return nil
			})
			if err != nil {
				return err
			}
			st, _ := store.Load()
			audit(st, "post draft", true, map[string]any{"name": name})
			out.P.Print("post draft", map[string]any{"name": name, "action": "created"}, nil)
			return nil
		},
	}
	c.Flags().StringVar(&title, "title", "", "post title")
	c.Flags().StringVar(&body, "body", "", "post body")
	c.Flags().StringVar(&bodyFile, "body-file", "", "read body from a file")
	c.Flags().StringVar(&collection, "collection", "", "collection name")
	c.Flags().StringVar(&media, "media", "", "comma-separated local media paths")
	c.Flags().StringVar(&tags, "tags", "", "comma-separated tags")
	return c
}

func postComposeCmd() *cobra.Command {
	// compose is an alias for draft with the same flags (person-friendly name)
	c := postDraftCmd()
	c.Use = "compose NAME"
	c.Short = "Compose a local post (same as draft)"
	return c
}

type postListResult struct {
	Posts []postRow `json:"posts"`
}

type postRow struct {
	Name       string   `json:"name"`
	Title      string   `json:"title,omitempty"`
	Collection string   `json:"collection,omitempty"`
	Media      []string `json:"media,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	Bytes      int      `json:"bytes"`
}

func (r postListResult) Text() string {
	if len(r.Posts) == 0 {
		return "No posts yet: kongkow post draft NAME --body '...'\n"
	}
	var b strings.Builder
	for _, p := range r.Posts {
		line := p.Name
		if p.Title != "" {
			line += "  " + p.Title
		} else {
			line += fmt.Sprintf("  %d bytes", p.Bytes)
		}
		if p.Collection != "" {
			line += "  [" + p.Collection + "]"
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

func postListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List local posts",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			st, err := store.Load()
			if err != nil {
				return err
			}
			r := postListResult{Posts: []postRow{}}
			for _, p := range st.SortedPosts() {
				r.Posts = append(r.Posts, postRow{
					Name: p.Name, Title: p.Title, Collection: p.Collection,
					Media: p.Media, Tags: p.Tags, Bytes: len(p.Body),
				})
			}
			out.P.Print("post list", r, nil)
			return nil
		},
	}
}

func postShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show NAME",
		Short: "Show one post",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			st, err := store.Load()
			if err != nil {
				return err
			}
			p := st.Posts[args[0]]
			if p == nil {
				return out.Errf(out.ExitNotFound, "POST_NOT_FOUND",
					"check the name; `kongkow post list` shows posts",
					"post %q is not in the store", args[0])
			}
			out.P.Print("post show", p, nil)
			return nil
		},
	}
}

func postEditCmd() *cobra.Command {
	var title, body, bodyFile, collection, media, tags string
	var setTitle, setBody, setCollection, setMedia, setTags bool
	c := &cobra.Command{
		Use:   "edit NAME",
		Short: "Edit a local post",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			setTitle = cmd.Flags().Changed("title")
			setBody = cmd.Flags().Changed("body") || cmd.Flags().Changed("body-file")
			setCollection = cmd.Flags().Changed("collection")
			setMedia = cmd.Flags().Changed("media")
			setTags = cmd.Flags().Changed("tags")
			text, err := readBodyFlag(body, bodyFile)
			if err != nil {
				return err
			}
			name := args[0]
			err = store.Update(func(s *store.State) error {
				p := s.Posts[name]
				if p == nil {
					return out.Errf(out.ExitNotFound, "POST_NOT_FOUND",
						"check the name; `kongkow post list` shows posts",
						"post %q is not in the store", name)
				}
				if setTitle {
					p.Title = title
				}
				if setBody {
					p.Body = text
				}
				if setCollection {
					if collection != "" && s.Collections[collection] == nil {
						return out.Errf(out.ExitNotFound, "COLLECTION_NOT_FOUND",
							"check the name; `kongkow collection list` shows collections",
							"collection %q is not in the store", collection)
					}
					p.Collection = collection
				}
				if setMedia {
					p.Media = splitCSV(media)
				}
				if setTags {
					p.Tags = splitCSV(tags)
				}
				p.UpdatedAt = time.Now().UTC()
				return nil
			})
			if err != nil {
				return err
			}
			st, _ := store.Load()
			audit(st, "post edit", true, map[string]any{"name": name})
			out.P.Print("post edit", map[string]any{"name": name, "action": "changed"}, nil)
			return nil
		},
	}
	c.Flags().StringVar(&title, "title", "", "post title")
	c.Flags().StringVar(&body, "body", "", "post body")
	c.Flags().StringVar(&bodyFile, "body-file", "", "read body from a file")
	c.Flags().StringVar(&collection, "collection", "", "collection name (empty clears)")
	c.Flags().StringVar(&media, "media", "", "comma-separated local media paths")
	c.Flags().StringVar(&tags, "tags", "", "comma-separated tags")
	return c
}

func postRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm NAME",
		Short: "Remove a local post",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			name := args[0]
			err := store.Update(func(s *store.State) error {
				if s.Posts[name] == nil {
					return out.Errf(out.ExitNotFound, "POST_NOT_FOUND",
						"check the name; `kongkow post list` shows posts",
						"post %q is not in the store", name)
				}
				delete(s.Posts, name)
				return nil
			})
			if err != nil {
				return err
			}
			st, _ := store.Load()
			audit(st, "post rm", true, map[string]any{"name": name})
			out.P.Print("post rm", map[string]any{"name": name, "action": "removed"}, nil)
			return nil
		},
	}
}

func postPublishCmd() *cobra.Command {
	var to string
	c := &cobra.Command{
		Use:   "publish NAME",
		Short: "Publish a post to a destination",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if to == "" {
				return out.Usage("need --to DEST")
			}
			st, err := store.Load()
			if err != nil {
				return err
			}
			p := st.Posts[args[0]]
			if p == nil {
				return out.Errf(out.ExitNotFound, "POST_NOT_FOUND",
					"check the name; `kongkow post list` shows posts",
					"post %q is not in the store", args[0])
			}
			d := st.Dests[to]
			if d == nil {
				return out.Errf(out.ExitNotFound, "DEST_NOT_FOUND",
					"check the name; `kongkow dest list` shows destinations",
					"destination %q is not in the store", to)
			}
			if err := protectDest(d, "publishing"); err != nil {
				return err
			}
			content := dest.Content{Name: p.Name, Title: p.Title, Body: p.Body, Media: p.Media}
			if err := checkLength(d, content); err != nil {
				return err
			}
			if g.dryRun {
				out.P.Print("post publish", map[string]any{
					"name": p.Name, "dest": d.Name, "kind": d.Kind, "action": "would_publish",
				}, nil)
				return nil
			}
			res, err := dest.Publish(context.Background(), d, content, vault.Resolve)
			if err != nil {
				audit(st, "post publish", false, map[string]any{"name": p.Name, "dest": d.Name, "error": err.Error()})
				return err
			}
			audit(st, "post publish", true, map[string]any{"name": p.Name, "dest": d.Name, "kind": d.Kind})
			out.P.Print("post publish", map[string]any{
				"name": p.Name, "dest": d.Name, "kind": d.Kind, "action": "published",
				"path": res.Path, "detail": res.Detail, "target": res.Target,
			}, nil)
			return nil
		},
	}
	c.Flags().StringVar(&to, "to", "", "destination name")
	return c
}

func postScheduleCmd() *cobra.Command {
	var to, at string
	c := &cobra.Command{
		Use:   "schedule NAME",
		Short: "Schedule a local publish job",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if to == "" || at == "" {
				return out.Usage("need --to DEST and --at TIME")
			}
			when, err := parseTime(at)
			if err != nil {
				return err
			}
			var jobID string
			err = store.Update(func(s *store.State) error {
				if s.Posts[args[0]] == nil {
					return out.Errf(out.ExitNotFound, "POST_NOT_FOUND",
						"check the name; `kongkow post list` shows posts",
						"post %q is not in the store", args[0])
				}
				d := s.Dests[to]
				if d == nil {
					return out.Errf(out.ExitNotFound, "DEST_NOT_FOUND",
						"check the name; `kongkow dest list` shows destinations",
						"destination %q is not in the store", to)
				}
				if err := protectDest(d, "scheduling a publish"); err != nil {
					return err
				}
				p := s.Posts[args[0]]
				if err := checkLength(d, dest.Content{Name: p.Name, Title: p.Title, Body: p.Body, Media: p.Media}); err != nil {
					return err
				}
				id, err := newJobID()
				if err != nil {
					return err
				}
				jobID = id
				now := time.Now().UTC()
				s.Jobs[id] = &store.Job{
					ID: id, Post: args[0], Dest: to, At: when.UTC(),
					Status: store.JobPending, CreatedAt: now, UpdatedAt: now,
				}
				return nil
			})
			if err != nil {
				return err
			}
			st, _ := store.Load()
			audit(st, "post schedule", true, map[string]any{"id": jobID, "post": args[0], "dest": to, "at": when.UTC()})
			out.P.Print("post schedule", map[string]any{
				"id": jobID, "post": args[0], "dest": to, "at": when.UTC(), "action": "scheduled",
			}, nil)
			return nil
		},
	}
	c.Flags().StringVar(&to, "to", "", "destination name")
	c.Flags().StringVar(&at, "at", "", "when to publish (RFC3339 or 2006-01-02 15:04)")
	return c
}

type jobRow struct {
	ID     string    `json:"id"`
	Post   string    `json:"post"`
	Dest   string    `json:"dest"`
	At     time.Time `json:"at"`
	Status string    `json:"status"`
	Result string    `json:"result,omitempty"`
}

type statusResult struct {
	Jobs    []jobRow `json:"jobs"`
	Ran     []jobRow `json:"ran,omitempty"`
	Pending int      `json:"pending"`
}

func (r statusResult) Text() string {
	var b strings.Builder
	if len(r.Ran) > 0 {
		b.WriteString("Ran:\n")
		for _, j := range r.Ran {
			fmt.Fprintf(&b, "  %s  %s → %s  %s  %s\n", j.ID, j.Post, j.Dest, j.Status, j.Result)
		}
	}
	if len(r.Jobs) == 0 {
		b.WriteString("No jobs.\n")
		return b.String()
	}
	for _, j := range r.Jobs {
		fmt.Fprintf(&b, "%s  %s → %s  %s  %s\n", j.ID, j.Post, j.Dest, j.At.Local().Format("2006-01-02 15:04"), j.Status)
	}
	return b.String()
}

func postStatusCmd() *cobra.Command {
	var run bool
	c := &cobra.Command{
		Use:   "status",
		Short: "Show scheduled publish jobs (optionally run due ones)",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			var ran []jobRow
			if run && !g.dryRun {
				var err error
				ran, err = runDueJobs()
				if err != nil {
					return err
				}
			}
			st, err := store.Load()
			if err != nil {
				return err
			}
			r := statusResult{Jobs: []jobRow{}, Ran: ran}
			for _, j := range st.SortedJobs() {
				r.Jobs = append(r.Jobs, jobRow{ID: j.ID, Post: j.Post, Dest: j.Dest, At: j.At, Status: j.Status, Result: j.Result})
				if j.Status == store.JobPending {
					r.Pending++
				}
			}
			if g.dryRun && run {
				out.P.Print("post status", map[string]any{"action": "would_run_due", "pending": r.Pending, "jobs": r.Jobs}, nil)
				return nil
			}
			out.P.Print("post status", r, nil)
			for _, j := range ran {
				if j.Status == store.JobFailed {
					return exitCode(1) // the envelope above lists which job failed and why
				}
			}
			return nil
		},
	}
	c.Flags().BoolVar(&run, "run", false, "publish jobs whose --at time has passed")
	return c
}

func runDueJobs() ([]jobRow, error) {
	st, err := store.Load()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	var due []string
	for _, j := range st.SortedJobs() {
		if j.Status == store.JobPending && !j.At.After(now) {
			due = append(due, j.ID)
		}
	}
	var ran []jobRow
	for _, id := range due {
		st, err = store.Load()
		if err != nil {
			return ran, err
		}
		j := st.Jobs[id]
		if j == nil {
			continue
		}
		p := st.Posts[j.Post]
		d := st.Dests[j.Dest]
		row := jobRow{ID: j.ID, Post: j.Post, Dest: j.Dest, At: j.At}
		if p == nil || d == nil {
			_ = store.Update(func(s *store.State) error {
				jj := s.Jobs[id]
				if jj == nil {
					return nil
				}
				jj.Status = store.JobFailed
				jj.Result = "post or destination missing"
				jj.UpdatedAt = time.Now().UTC()
				return nil
			})
			row.Status = store.JobFailed
			row.Result = "post or destination missing"
			ran = append(ran, row)
			continue
		}
		if err := protectDest(d, "publishing"); err != nil {
			row.Status = store.JobFailed
			row.Result = err.Error()
			ran = append(ran, row)
			continue
		}
		claimed := false
		if err := store.Update(func(s *store.State) error {
			jj := s.Jobs[id]
			if jj == nil || jj.Status != store.JobPending {
				return nil
			}
			jj.Status = store.JobRunning
			jj.UpdatedAt = time.Now().UTC()
			claimed = true
			return nil
		}); err != nil {
			return ran, err
		}
		if !claimed {
			continue
		}
		res, pubErr := dest.Publish(context.Background(), d, dest.Content{
			Name: p.Name, Title: p.Title, Body: p.Body, Media: p.Media,
		}, vault.Resolve)
		_ = store.Update(func(s *store.State) error {
			jj := s.Jobs[id]
			if jj == nil {
				return nil
			}
			jj.UpdatedAt = time.Now().UTC()
			if pubErr != nil {
				jj.Status = store.JobFailed
				jj.Result = pubErr.Error()
			} else {
				jj.Status = store.JobDone
				jj.Result = res.Detail
				if res.Path != "" {
					jj.Result = res.Path
				}
			}
			return nil
		})
		if pubErr != nil {
			row.Status = store.JobFailed
			row.Result = pubErr.Error()
			audit(st, "post status --run", false, map[string]any{"id": id, "error": pubErr.Error()})
		} else {
			row.Status = store.JobDone
			row.Result = res.Detail
			if res.Path != "" {
				row.Result = res.Path
			}
			audit(st, "post status --run", true, map[string]any{"id": id, "dest": d.Name})
		}
		ran = append(ran, row)
	}
	return ran, nil
}

func newJobID() (string, error) {
	id, err := storeNewID()
	if err != nil {
		return "", err
	}
	if len(id) > 12 {
		id = id[:12]
	}
	return "job_" + id, nil
}

func postCancelCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cancel JOB",
		Short: "Cancel a pending or stuck scheduled publish",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			id := args[0]
			err := store.Update(func(s *store.State) error {
				j := s.Jobs[id]
				if j == nil {
					return out.Errf(out.ExitNotFound, "JOB_NOT_FOUND",
						"check the id; `kongkow post status` shows jobs",
						"job %q is not in the store", id)
				}
				if j.Status != store.JobPending && !isStale(j) {
					return out.Errf(out.ExitConflict, "JOB_NOT_PENDING",
						"only a pending job, or one stuck running (see `kongkow doctor`), can be cancelled",
						"job %q is %s", id, j.Status)
				}
				j.Status = store.JobCancelled
				j.UpdatedAt = time.Now().UTC()
				return nil
			})
			if err != nil {
				return err
			}
			st, _ := store.Load()
			audit(st, "post cancel", true, map[string]any{"id": id})
			out.P.Print("post cancel", map[string]any{"id": id, "action": "cancelled"}, nil)
			return nil
		},
	}
}

// staleAfter is how long a job may stay running before doctor calls it
// stuck. A publish takes seconds; this leaves room for a slow network.
const staleAfter = 10 * time.Minute

func isStale(j *store.Job) bool {
	return j.Status == store.JobRunning && time.Since(j.UpdatedAt) > staleAfter
}

func staleJobs(st *store.State) []string {
	var ids []string
	for _, j := range st.SortedJobs() {
		if isStale(j) {
			ids = append(ids, j.ID)
		}
	}
	return ids
}

// checkLength refuses a post that is too long for d's network, before
// anything is sent or scheduled.
func checkLength(d *store.Dest, c dest.Content) error {
	if err := dest.CheckLength(d, c); err != nil {
		return out.Errf(out.ExitUsage, "POST_TOO_LONG",
			"shorten the post with `kongkow post edit NAME`", "%v", err)
	}
	return nil
}
