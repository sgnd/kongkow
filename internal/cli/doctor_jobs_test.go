package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/sgnd/kongkow/internal/dest"
	"github.com/sgnd/kongkow/internal/store"
)

func stateWithJobs(jobs ...*store.Job) *store.State {
	st := &store.State{Posts: map[string]*store.Post{}, Dests: map[string]*store.Dest{}, Jobs: map[string]*store.Job{}}
	st.Dests["m"] = &store.Dest{Name: "m", Kind: store.DestMastodon}
	st.Dests["t"] = &store.Dest{Name: "t", Kind: store.DestTelegram}
	st.Dests["unused"] = &store.Dest{Name: "unused", Kind: store.DestBluesky}
	for _, j := range jobs {
		st.Jobs[j.ID] = j
	}
	return st
}

func TestTokenChecksCoverEveryDestinationWithAPendingJob(t *testing.T) {
	st := stateWithJobs(
		&store.Job{ID: "1", Dest: "m", Status: store.JobPending},
		&store.Job{ID: "2", Dest: "m", Status: store.JobPending}, // same dest: one test
		&store.Job{ID: "3", Dest: "t", Status: store.JobPending},
		&store.Job{ID: "4", Dest: "unused", Status: store.JobDone}, // done: not tested
	)
	var asked []string
	test := func(_ context.Context, d *store.Dest) (*dest.Result, error) {
		asked = append(asked, d.Name)
		if d.Name == "t" {
			return nil, errors.New("telegram: Unauthorized")
		}
		return &dest.Result{Detail: "account @me"}, nil
	}
	checks := tokenChecks(context.Background(), st, test)
	if strings.Join(asked, ",") != "m,t" {
		t.Fatalf("tested %v, want m then t once each", asked)
	}
	if len(checks) != 2 {
		t.Fatalf("checks %+v", checks)
	}
	if checks[0].Name != "token m" || !checks[0].OK {
		t.Errorf("m: %+v", checks[0])
	}
	if checks[1].Name != "token t" || checks[1].OK || !strings.Contains(checks[1].Detail, "Unauthorized") {
		t.Errorf("t: %+v", checks[1])
	}
}

func TestTokenChecksSkipWhenNothingIsScheduled(t *testing.T) {
	checks := tokenChecks(context.Background(), stateWithJobs(), func(context.Context, *store.Dest) (*dest.Result, error) {
		t.Fatal("no job is pending, so nothing should be tested")
		return nil, nil
	})
	if len(checks) != 1 || checks[0].Name != "tokens" || !checks[0].OK {
		t.Errorf("checks %+v", checks)
	}
}

func TestTokenCheckForAMissingDestination(t *testing.T) {
	st := stateWithJobs(&store.Job{ID: "1", Dest: "gone", Status: store.JobPending})
	checks := tokenChecks(context.Background(), st, func(context.Context, *store.Dest) (*dest.Result, error) {
		t.Fatal("a missing destination cannot be tested")
		return nil, nil
	})
	if len(checks) != 1 || checks[0].OK || !strings.Contains(checks[0].Detail, "gone") {
		t.Errorf("checks %+v", checks)
	}
}

func TestFailedJobsCheckListsRecentFailures(t *testing.T) {
	now := time.Now()
	st := stateWithJobs(
		&store.Job{ID: "new", Dest: "m", Status: store.JobFailed, Result: "401", UpdatedAt: now.Add(-time.Hour)},
		&store.Job{ID: "old", Dest: "m", Status: store.JobFailed, Result: "401", UpdatedAt: now.Add(-8 * 24 * time.Hour)},
		&store.Job{ID: "ok", Dest: "m", Status: store.JobDone, UpdatedAt: now},
	)
	c := failedJobsCheck(st, now)
	if c.OK || !strings.Contains(c.Detail, "new") || strings.Contains(c.Detail, "old") {
		t.Errorf("check %+v", c)
	}
	if c := failedJobsCheck(stateWithJobs(), now); !c.OK {
		t.Errorf("no failures should be ok: %+v", c)
	}
}

func TestDoctorFailsWhenAScheduledTokenIsRejected(t *testing.T) {
	setupJobs(t)
	run(t, 0, "post", "schedule", "n1", "--to", "f1", "--at", time.Now().Add(time.Hour).Format(time.RFC3339))

	old := testDest
	t.Cleanup(func() { testDest = old })
	testDest = func(context.Context, *store.Dest) (*dest.Result, error) { return &dest.Result{Detail: "ok"}, nil }
	run(t, 0, "doctor")

	testDest = func(context.Context, *store.Dest) (*dest.Result, error) { return nil, errors.New("rejected") }
	run(t, 1, "doctor")
	run(t, 0, "doctor", "--offline") // no network, no token test
}

func TestNoticeForAFailedScheduledJob(t *testing.T) {
	var buf bytes.Buffer
	printNotices(&buf, []jobRow{
		{ID: "j1", Post: "launch", Dest: "m", Status: store.JobDone},
		{ID: "j2", Post: "launch", Dest: "t", Status: store.JobFailed, Result: "telegram: chat not found\x1b[2J"},
	})
	got := buf.String()
	for _, want := range []string{"NOTICE", "j2", "launch", "t", "chat not found", "kongkow post status"} {
		if !strings.Contains(got, want) {
			t.Errorf("notice lacks %q: %q", want, got)
		}
	}
	if strings.Contains(got, "j1") || strings.Contains(got, "\x1b") {
		t.Errorf("a done job or a control character is in the notice: %q", got)
	}
	buf.Reset()
	printNotices(&buf, nil)
	if buf.Len() != 0 {
		t.Error("nothing failed, so nothing is written")
	}
}
