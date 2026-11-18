package cli

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/sgnd/kongkow/internal/out"
	"github.com/sgnd/kongkow/internal/store"
)

const slotsHelp = "daily post times for `post queue`, local, like 09:00,13:00,18:30"

// queueDays is how far ahead `post queue` looks for a free slot.
const queueDays = 60

// parseSlots turns "13:00, 9:00" into sorted, unique ["09:00","13:00"].
func parseSlots(s string) ([]string, error) {
	seen := map[string]bool{}
	var slots []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		t, err := time.Parse("15:04", p)
		if err != nil {
			return nil, out.Usage("slot %q is not a time like 09:00", p)
		}
		v := t.Format("15:04")
		if !seen[v] {
			seen[v] = true
			slots = append(slots, v)
		}
	}
	sort.Strings(slots)
	return slots, nil
}

// nextSlot is the first slot of d after now that no pending or running
// job of d already uses.
func nextSlot(s *store.State, d *store.Dest, now time.Time) (time.Time, error) {
	if len(d.Slots) == 0 {
		return time.Time{}, out.Errf(out.ExitUsage, "NO_SLOTS",
			fmt.Sprintf("set times with `kongkow dest edit %s --slots 09:00,13:00`", d.Name),
			"destination %q has no queue slots", d.Name)
	}
	taken := map[time.Time]bool{}
	for _, j := range s.Jobs {
		if j.Dest == d.Name && (j.Status == store.JobPending || j.Status == store.JobRunning) {
			taken[j.At.UTC()] = true
		}
	}
	local := now.Local()
	for day := 0; day <= queueDays; day++ {
		date := local.AddDate(0, 0, day)
		for _, slot := range d.Slots {
			t, _ := time.Parse("15:04", slot)
			at := time.Date(date.Year(), date.Month(), date.Day(), t.Hour(), t.Minute(), 0, 0, time.Local)
			if at.After(now) && !taken[at.UTC()] {
				return at, nil
			}
		}
	}
	return time.Time{}, out.Errf(out.ExitConflict, "QUEUE_FULL",
		"add slots or cancel jobs; `kongkow post status` shows them",
		"every slot of %q in the next %d days is taken", d.Name, queueDays)
}

func postQueueCmd() *cobra.Command {
	var to string
	c := &cobra.Command{
		Use:   "queue NAME",
		Short: "Schedule a post at the next free slot of each destination",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			names, err := destNames(to)
			if err != nil {
				return err
			}
			var rows []map[string]any
			err = store.Update(func(s *store.State) error {
				p := s.Posts[args[0]]
				if p == nil {
					return out.Errf(out.ExitNotFound, "POST_NOT_FOUND",
						"check the name; `kongkow post list` shows posts",
						"post %q is not in the store", args[0])
				}
				content := contentOf(p)
				dests, err := checkDests(s, names, content, "scheduling a publish")
				if err != nil {
					return err
				}
				now := time.Now()
				rows, err = addJobs(s, args[0], dests, func(d *store.Dest) (time.Time, error) {
					return nextSlot(s, d, now)
				})
				return err
			})
			if err != nil {
				return err
			}
			printJobs("post queue", rows)
			return nil
		},
	}
	c.Flags().StringVar(&to, "to", "", "destination name, or several separated by commas")
	return c
}
