package cli

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

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

func doctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check kongkow's setup and store",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			r := doctorReport{optional: map[string]bool{
				"posts": true, "dests": true, "collections": true, "bookmarks": true, "jobs": true,
			}}
			add := func(name string, ok bool, detail string) {
				r.Checks = append(r.Checks, doctorCheck{name, ok, detail})
			}
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
}

func emptyOrCount(n int, singular string) string {
	if n == 0 {
		return "none yet"
	}
	return countOf(n, singular)
}
