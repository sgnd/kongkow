package cli

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/sgnd/kongkow/internal/dest"
	"github.com/sgnd/kongkow/internal/out"
	"github.com/sgnd/kongkow/internal/store"
	"github.com/sgnd/kongkow/internal/vault"
)

type doctorCheck struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

type doctorReport struct {
	Checks   []doctorCheck `json:"checks"`
	optional map[string]bool
}

func (r doctorReport) Text() string {
	var b strings.Builder
	fails, warns := 0, 0
	for _, c := range r.Checks {
		mark := "✓"
		switch {
		case c.OK:
		case r.optional[c.Name]:
			warns++
			mark = "!"
		default:
			fails++
			mark = "✗"
		}
		line := out.Marks(mark) + " " + c.Name
		if c.Detail != "" {
			line += ": " + c.Detail
		}
		b.WriteString(line + "\n")
	}
	switch {
	case fails == 0 && warns == 0:
		b.WriteString("\nAll good: kongkow is ready.\n")
	default:
		fmt.Fprintf(&b, "\n%s to fix, %s.\n", countOf(fails, "problem"), countOf(warns, "warning"))
	}
	return b.String()
}

// testDest checks one destination's token without posting. Tests replace it.
var testDest = func(ctx context.Context, d *store.Dest) (*dest.Result, error) {
	return dest.Test(ctx, d, vault.Resolve)
}

const (
	tokenTestTimeout = 15 * time.Second
	failedJobsWindow = 7 * 24 * time.Hour
)

// tokenChecks tests the token of every destination that has a pending job,
// once each, so a dead token shows up before the job is due and not as a
// silent failure at the hour of the post.
func tokenChecks(ctx context.Context, st *store.State, test func(context.Context, *store.Dest) (*dest.Result, error)) []doctorCheck {
	seen := map[string]bool{}
	var names []string
	for _, j := range st.Jobs {
		if j.Status == store.JobPending && !seen[j.Dest] {
			seen[j.Dest] = true
			names = append(names, j.Dest)
		}
	}
	if len(names) == 0 {
		return []doctorCheck{{"tokens", true, "no scheduled jobs to check"}}
	}
	sort.Strings(names)
	var checks []doctorCheck
	for _, n := range names {
		d := st.Dests[n]
		if d == nil {
			checks = append(checks, doctorCheck{"token " + n, false, "a scheduled job names the destination " + out.SafeText(n) + ", which no longer exists; cancel the job with `kongkow post cancel JOB`"})
			continue
		}
		cctx, cancel := context.WithTimeout(ctx, tokenTestTimeout)
		res, err := test(cctx, d)
		cancel()
		if err != nil {
			checks = append(checks, doctorCheck{"token " + n, false, out.SafeText(dest.Redact(err.Error())) + ": fix the token with `kongkow secret set`, then run doctor again"})
			continue
		}
		checks = append(checks, doctorCheck{"token " + n, true, out.SafeText(res.Detail)})
	}
	return checks
}

// failedJobsCheck lists the jobs that failed in the last week, so a failure
// seen only in a loop's log is still there when the person looks.
func failedJobsCheck(st *store.State, now time.Time) doctorCheck {
	var failed []string
	for _, j := range st.SortedJobs() {
		if j.Status == store.JobFailed && now.Sub(j.UpdatedAt) < failedJobsWindow {
			failed = append(failed, j.ID+" ("+out.SafeText(dest.Redact(j.Result))+")")
		}
	}
	if len(failed) == 0 {
		return doctorCheck{"failed jobs", true, "none in the last 7 days"}
	}
	return doctorCheck{"failed jobs", false, fmt.Sprintf("%s failed in the last 7 days: %s; see `kongkow post status`", countOf(len(failed), "job"), strings.Join(failed, ", "))}
}

// printNotices writes one line to w for every job that failed in a run.
func printNotices(w io.Writer, ran []jobRow) {
	for _, j := range ran {
		if j.Status == store.JobFailed {
			fmt.Fprintf(w, "NOTICE: job %s (%s to %s) failed: %s. See `kongkow post status`, fix it, and schedule the post again.\n",
				j.ID, out.SafeText(j.Post), out.SafeText(j.Dest), out.SafeText(dest.Redact(j.Result)))
		}
	}
}

func doctorCmd() *cobra.Command {
	var offline bool
	c := &cobra.Command{
		Use:   "doctor",
		Short: "Check kongkow's setup, store and the tokens of scheduled posts",
		Long:  `Check kongkow's setup and store, and test the token of every destination that has a scheduled post, so a dead token shows up before the post is due. The token test asks each destination's service, so it needs the network; --offline skips it. A job that failed in the last 7 days is listed too.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			r := doctorReport{optional: map[string]bool{
				"posts": true, "dests": true, "collections": true, "bookmarks": true, "jobs": true, "failed jobs": true,
			}}
			add := func(name string, ok bool, detail string) {
				r.Checks = append(r.Checks, doctorCheck{name, ok, detail})
			}
			scrubSavedResults()
			st, err := store.Load()
			add("state", err == nil, store.Home())
			if err != nil {
				out.P.Print("doctor", r, toErr(err))
				return exitCode(1)
			}
			if _, err := store.DB(true); err != nil {
				add("store", false, err.Error())
				out.P.Print("doctor", r, nil)
				return exitCode(1)
			}
			add("store", true, store.Path(store.StoreFile))
			str, err := vault.Check()
			switch {
			case err != nil:
				add("vault", false, err.Error())
			case str.NoVaultYet:
				add("vault", true, "no secrets yet")
			default:
				detail := "mode " + str.Mode
				ok := true
				if str.ShortPass && str.Mode == "passphrase" {
					detail += " (passphrase shorter than 15 characters)"
					ok = false
					r.optional["vault"] = true
				}
				add("vault", ok, detail)
			}
			add("posts", true, emptyOrCount(len(st.Posts), "post"))
			add("dests", true, emptyOrCount(len(st.Dests), "destination"))
			add("collections", true, emptyOrCount(len(st.Collections), "collection"))
			add("bookmarks", true, emptyOrCount(len(st.Bookmarks), "bookmark"))
			if stale := staleJobs(st); len(stale) > 0 {
				add("jobs", false, fmt.Sprintf("%s stuck running (%s): a run stopped mid-publish; "+
					"check the destination, then `kongkow post cancel JOB` and schedule again if it was not sent",
					countOf(len(stale), "job"), strings.Join(stale, ", ")))
			} else {
				add("jobs", true, "none stuck")
			}
			add("platform", true, runtime.GOOS+"/"+runtime.GOARCH)
			fc := failedJobsCheck(st, time.Now())
			add(fc.Name, fc.OK, fc.Detail)
			if offline {
				add("tokens", true, "not tested (--offline)")
			} else {
				for _, tc := range tokenChecks(cmd.Context(), st, testDest) {
					add(tc.Name, tc.OK, tc.Detail)
				}
			}
			fails := 0
			for _, c := range r.Checks {
				if !c.OK && !r.optional[c.Name] {
					fails++
				}
			}
			out.P.Print("doctor", r, nil)
			if fails > 0 {
				return exitCode(1)
			}
			return nil
		},
	}
	c.Flags().BoolVar(&offline, "offline", false, "do not test the tokens of scheduled posts (no network)")
	return c
}

func emptyOrCount(n int, singular string) string {
	if n == 0 {
		return "none yet"
	}
	return countOf(n, singular)
}
