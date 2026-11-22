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

// A fake token, shaped like a Telegram bot token.
const fakeBotToken = "123456789:AAFAKEFAKEFAKEFAKEFAKEFAKEFAKEFAKE_-x"

var leaky = `Post "https://api.telegram.org/bot` + fakeBotToken + `/sendMessage": dial tcp 127.0.0.1:1: connect: connection refused`

func noToken(t *testing.T, where, text string) {
	t.Helper()
	if strings.Contains(text, "AAFAKE") || strings.Contains(text, fakeBotToken) {
		t.Errorf("%s holds the token: %q", where, text)
	}
}

func TestApplyResultKeepsTokensOutOfTheJob(t *testing.T) {
	j := &store.Job{ID: "j1", Status: store.JobRunning}
	applyResult(j, nil, errors.New(leaky), time.Now())
	noToken(t, "job Result", j.Result)
	if !strings.Contains(j.Result, "sendMessage") {
		t.Errorf("the failure should still say what failed: %q", j.Result)
	}
}

// Results saved by an earlier version may hold a token already: every place
// that shows one must hide it.
func TestSavedResultsAreHiddenWhereTheyAreShown(t *testing.T) {
	noToken(t, "post status line", jobLine(jobRow{ID: "j", Post: "p", Dest: "t", Status: store.JobFailed, Result: leaky}, true))

	var notice bytes.Buffer
	printNotices(&notice, []jobRow{{ID: "j", Post: "p", Dest: "t", Status: store.JobFailed, Result: leaky}})
	noToken(t, "NOTICE", notice.String())

	st := stateWithJobs(&store.Job{ID: "old", Dest: "t", Status: store.JobFailed, Result: leaky, UpdatedAt: time.Now()})
	noToken(t, "doctor failed-jobs check", failedJobsCheck(st, time.Now()).Detail)

	st = stateWithJobs(&store.Job{ID: "1", Dest: "t", Status: store.JobPending})
	checks := tokenChecks(context.Background(), st, func(context.Context, *store.Dest) (*dest.Result, error) {
		return nil, errors.New(leaky)
	})
	noToken(t, "doctor token check", checks[0].Detail)

	noToken(t, "job row", rowOf(&store.Job{ID: "j", Status: store.JobFailed, Result: leaky}).Result)
}

func TestScrubSavedResults(t *testing.T) {
	setupJobs(t)
	err := store.Update(func(s *store.State) error {
		s.Jobs["old"] = &store.Job{ID: "old", Post: "n1", Dest: "f1", Status: store.JobFailed, Result: leaky, UpdatedAt: time.Now()}
		s.Jobs["fine"] = &store.Job{ID: "fine", Post: "n1", Dest: "f1", Status: store.JobDone, Result: "/out/a.md", UpdatedAt: time.Now()}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	run(t, 0, "post", "status") // looking at the jobs removes the token from the saved state
	st, _ := store.Load()
	noToken(t, "saved job Result", st.Jobs["old"].Result)
	if st.Jobs["fine"].Result != "/out/a.md" {
		t.Errorf("a clean result was changed: %q", st.Jobs["fine"].Result)
	}
	if !strings.Contains(st.Jobs["old"].Result, "sendMessage") {
		t.Errorf("the scrubbed result lost its meaning: %q", st.Jobs["old"].Result)
	}
}
