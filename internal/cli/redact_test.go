package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/sgnd/kongkow/internal/dest"
	"github.com/sgnd/kongkow/internal/out"
	"github.com/sgnd/kongkow/internal/sealdb"
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

// Audit entries written by v0.0.1 to v0.1.1 may hold a token in "error".
func TestAuditHidesAndScrubsOldTokens(t *testing.T) {
	setupJobs(t)
	if err := writeAudit(map[string]any{"ts": "2026-11-18T00:00:00Z", "command": "post publish", "ok": false, "error": leaky}); err != nil {
		t.Fatal(err)
	}
	if err := writeAudit(map[string]any{"ts": "2026-11-18T00:00:01Z", "command": "post draft", "ok": true}); err != nil {
		t.Fatal(err)
	}

	env := runJSON(t, 0, "audit", "--limit", "0")
	b, _ := json.Marshal(env)
	noToken(t, "audit JSON", string(b))
	if !strings.Contains(string(b), "sendMessage") {
		t.Errorf("the entry lost its meaning: %s", b)
	}

	resetOut()
	var text bytes.Buffer
	out.P.W = &text
	out.P.Format = "text"
	if code := executeArgs([]string{"audit", "--limit", "0", "-o", "text"}); code != 0 {
		t.Fatalf("audit text exit %d", code)
	}
	noToken(t, "audit text", text.String())

	// the stored entries are clean too, and the log still reads as a whole
	entries, err := readAudit(0)
	if err != nil {
		t.Fatalf("audit log broken after the scrub: %v", err)
	}
	for _, e := range entries {
		raw, _ := json.Marshal(e.Rec)
		noToken(t, "stored audit entry", string(raw))
	}
	if len(entries) < 2 {
		t.Errorf("entries %d", len(entries))
	}
}

func TestAuditTextShowsTheError(t *testing.T) {
	r := auditResult{Entries: []auditEntry{{Seq: 1, Rec: map[string]any{"command": "post publish", "ok": false, "error": leaky}}}}
	got := r.Text()
	noToken(t, "audit text", got)
	if !strings.Contains(got, "sendMessage") {
		t.Errorf("a failed entry should say why: %q", got)
	}
}

// Audit LOW 1: a rewrite leaves one normal entry that says it happened.
func TestAuditRedactLeavesOneTrace(t *testing.T) {
	setupJobs(t)
	if err := writeAudit(map[string]any{"command": "post draft", "ok": true}); err != nil {
		t.Fatal(err)
	}
	if err := writeAudit(map[string]any{"command": "post publish", "ok": false, "error": leaky}); err != nil {
		t.Fatal(err)
	}
	runJSON(t, 0, "audit", "--limit", "0")
	runJSON(t, 0, "audit", "--limit", "0")
	entries, err := readAudit(0)
	if err != nil {
		t.Fatal(err)
	}
	var leakSeq int64
	for _, e := range entries {
		if e.Rec["command"] == "post publish" {
			leakSeq = e.Seq
		}
	}
	traces := 0
	for _, e := range entries {
		if e.Rec["command"] == "audit redact" {
			traces++
			if e.Rec["entries"] != float64(1) || e.Rec["from_seq"] != float64(leakSeq) {
				t.Errorf("trace %v", e.Rec)
			}
		}
	}
	if traces != 1 {
		t.Fatalf("%d redact traces, want 1", traces)
	}
}

// Audit LOW 3: the scrub runs once per store, not on every command.
func TestAuditScrubRunsOnce(t *testing.T) {
	setupJobs(t)
	if err := writeAudit(map[string]any{"command": "post draft", "ok": true}); err != nil {
		t.Fatal(err)
	}
	runJSON(t, 0, "audit", "--limit", "0") // nothing to clean; marks the store done
	if err := writeAudit(map[string]any{"command": "post publish", "ok": false, "error": leaky}); err != nil {
		t.Fatal(err)
	}
	runJSON(t, 0, "audit", "--limit", "0")
	// read the stored rows directly: readAudit hides tokens on display
	d, err := store.DB(false)
	if err != nil {
		t.Fatal(err)
	}
	var last string
	if err := d.View(context.Background(), func(tx *sealdb.Tx) error {
		rows, err := tx.Audit(0, 1<<30)
		if len(rows) > 0 {
			last = string(rows[len(rows)-1].Data)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(last, fakeBotToken) {
		t.Fatal("the scrub ran a second time")
	}
}
