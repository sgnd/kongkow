package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sgnd/kongkow/internal/out"
	"github.com/sgnd/kongkow/internal/store"
)

// setupJobs makes a post "n1" and a file destination "f1" in a fresh home.
func setupJobs(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("KONGKOW_HOME", home)
	store.Close()
	t.Cleanup(store.Close)
	outDir := filepath.Join(home, "out")
	if err := os.MkdirAll(outDir, 0o700); err != nil {
		t.Fatal(err)
	}
	run(t, 0, "post", "draft", "n1", "--body", "hi")
	run(t, 0, "dest", "add", "f1", "--kind", "file", "--target", outDir)
	return outDir
}

func run(t *testing.T, want int, args ...string) {
	t.Helper()
	resetOut()
	if code := executeArgs(append(args, "-o", "json")); code != want {
		t.Fatalf("%v: want exit %d, got %d", args, want, code)
	}
}

func onlyJob(t *testing.T) *store.Job {
	t.Helper()
	st, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Jobs) != 1 {
		t.Fatalf("want 1 job, got %d", len(st.Jobs))
	}
	for _, j := range st.Jobs {
		return j
	}
	return nil
}

func TestScheduleRunDueJob(t *testing.T) {
	outDir := setupJobs(t)
	past := time.Now().Add(-time.Hour).Format(time.RFC3339)
	run(t, 0, "post", "schedule", "n1", "--to", "f1", "--at", past)
	run(t, 0, "post", "status", "--run")
	if j := onlyJob(t); j.Status != store.JobDone {
		t.Fatalf("status %q, result %q", j.Status, j.Result)
	}
	// A second run must not publish the done job again.
	run(t, 0, "post", "status", "--run")
	ents, _ := os.ReadDir(outDir)
	if len(ents) != 1 {
		t.Fatalf("want 1 published file, got %d", len(ents))
	}
}

func TestFutureJobNotRun(t *testing.T) {
	outDir := setupJobs(t)
	future := time.Now().Add(time.Hour).Format(time.RFC3339)
	run(t, 0, "post", "schedule", "n1", "--to", "f1", "--at", future)
	run(t, 0, "post", "status", "--run")
	if j := onlyJob(t); j.Status != store.JobPending {
		t.Fatalf("status %q", j.Status)
	}
	if ents, _ := os.ReadDir(outDir); len(ents) != 0 {
		t.Fatalf("published early: %d files", len(ents))
	}
}

func TestRunningJobNotClaimedTwice(t *testing.T) {
	outDir := setupJobs(t)
	past := time.Now().Add(-time.Hour).Format(time.RFC3339)
	run(t, 0, "post", "schedule", "n1", "--to", "f1", "--at", past)
	id := onlyJob(t).ID
	// Another run has claimed it.
	if err := store.Update(func(s *store.State) error {
		s.Jobs[id].Status = store.JobRunning
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	run(t, 0, "post", "status", "--run")
	if ents, _ := os.ReadDir(outDir); len(ents) != 0 {
		t.Fatalf("running job was published again: %d files", len(ents))
	}
}

func TestCancelJob(t *testing.T) {
	outDir := setupJobs(t)
	past := time.Now().Add(-time.Hour).Format(time.RFC3339)
	run(t, 0, "post", "schedule", "n1", "--to", "f1", "--at", past)
	id := onlyJob(t).ID
	run(t, 0, "post", "cancel", id)
	run(t, out.ExitConflict, "post", "cancel", id)
	run(t, out.ExitNotFound, "post", "cancel", "job_missing")
	run(t, 0, "post", "status", "--run")
	if j := onlyJob(t); j.Status != store.JobCancelled {
		t.Fatalf("status %q", j.Status)
	}
	if ents, _ := os.ReadDir(outDir); len(ents) != 0 {
		t.Fatalf("cancelled job was published: %d files", len(ents))
	}
}

func TestJobFailsWhenDestRemoved(t *testing.T) {
	setupJobs(t)
	past := time.Now().Add(-time.Hour).Format(time.RFC3339)
	run(t, 0, "post", "schedule", "n1", "--to", "f1", "--at", past)
	run(t, 0, "dest", "rm", "f1")
	run(t, 1, "post", "status", "--run")
	if j := onlyJob(t); j.Status != store.JobFailed {
		t.Fatalf("status %q", j.Status)
	}
}

func TestStuckJobInDoctorAndCancel(t *testing.T) {
	setupJobs(t)
	past := time.Now().Add(-time.Hour).Format(time.RFC3339)
	run(t, 0, "post", "schedule", "n1", "--to", "f1", "--at", past)
	id := onlyJob(t).ID
	setJob := func(status string, age time.Duration) {
		if err := store.Update(func(s *store.State) error {
			s.Jobs[id].Status = status
			s.Jobs[id].UpdatedAt = time.Now().Add(-age).UTC()
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	// A job that just started running is not stuck and cannot be cancelled.
	setJob(store.JobRunning, time.Minute)
	run(t, out.ExitConflict, "post", "cancel", id)
	// Doctor warns about a stuck job but still exits 0.
	setJob(store.JobRunning, time.Hour)
	env := runJSON(t, 0, "doctor")
	if !strings.Contains(fmt.Sprint(env["data"]), id) {
		t.Fatalf("doctor misses stuck job: %v", env["data"])
	}
	run(t, 0, "post", "cancel", id)
	if j := onlyJob(t); j.Status != store.JobCancelled {
		t.Fatalf("status %q", j.Status)
	}
}

func TestTooLongForBluesky(t *testing.T) {
	setupJobs(t)
	run(t, 0, "post", "draft", "long", "--body", strings.Repeat("a", 301))
	run(t, 0, "dest", "add", "bs", "--kind", "bluesky", "--target", "me.bsky.social", "--secret-ref", "secret:bs")
	run(t, out.ExitUsage, "post", "publish", "long", "--to", "bs", "--dry-run")
	run(t, out.ExitUsage, "post", "schedule", "long", "--to", "bs", "--at", "2030-01-01 10:00")
	run(t, out.ExitUsage, "dest", "add", "m", "--kind", "mastodon", "--target", "http://mastodon.social", "--secret-ref", "secret:m")
}

func TestPublishAndScheduleToMany(t *testing.T) {
	outDir := setupJobs(t)
	other := filepath.Join(t.TempDir(), "other")
	run(t, 0, "dest", "add", "f2", "--kind", "file", "--target", other)

	// An unknown name stops everything before anything is sent.
	run(t, out.ExitNotFound, "post", "publish", "n1", "--to", "f1,nope")
	if ents, _ := os.ReadDir(outDir); len(ents) != 0 {
		t.Fatalf("sent before checking all destinations: %d files", len(ents))
	}

	env := runJSON(t, 0, "post", "publish", "n1", "--to", "f1, f2,f1")
	rows, _ := env["data"].(map[string]any)["results"].([]any)
	if len(rows) != 2 {
		t.Fatalf("want 2 results, got %v", env["data"])
	}
	for _, dir := range []string{outDir, other} {
		if ents, _ := os.ReadDir(dir); len(ents) != 1 {
			t.Fatalf("%s: want 1 file, got %d", dir, len(ents))
		}
	}

	past := time.Now().Add(-time.Hour).Format(time.RFC3339)
	run(t, 0, "post", "schedule", "n1", "--to", "f1,f2", "--at", past)
	st, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Jobs) != 2 {
		t.Fatalf("want 2 jobs, got %d", len(st.Jobs))
	}
	run(t, out.ExitUsage, "post", "publish", "n1", "--to", " , ")
}

func TestWatchRunsDueJobs(t *testing.T) {
	outDir := setupJobs(t)
	soon := time.Now().Add(300 * time.Millisecond).Format(time.RFC3339Nano)
	run(t, 0, "post", "schedule", "n1", "--to", "f1", "--at", soon)
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	if err := watchJobs(ctx, 100*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if j := onlyJob(t); j.Status != store.JobDone {
		t.Fatalf("status %q", j.Status)
	}
	if ents, _ := os.ReadDir(outDir); len(ents) != 1 {
		t.Fatalf("want 1 file, got %d", len(ents))
	}
}

func TestWatchFlagRules(t *testing.T) {
	setupJobs(t)
	run(t, out.ExitUsage, "post", "status", "--every", "1m")
	run(t, out.ExitUsage, "post", "status", "--run", "--every", "1s")
	run(t, out.ExitUsage, "post", "status", "--run", "--every", "1m", "--dry-run")
}

func TestMediaCheckedBeforeSend(t *testing.T) {
	setupJobs(t)
	txt := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(txt, []byte("not an image"), 0o600); err != nil {
		t.Fatal(err)
	}
	run(t, 0, "post", "draft", "pic", "--body", "hi", "--media", txt)
	run(t, 0, "dest", "add", "bs", "--kind", "bluesky", "--target", "me.bsky.social", "--secret-ref", "secret:bs")
	run(t, out.ExitUsage, "post", "publish", "pic", "--to", "bs", "--dry-run")
	run(t, out.ExitUsage, "post", "schedule", "pic", "--to", "bs", "--at", "2030-01-01 10:00")
	run(t, 0, "post", "publish", "pic", "--to", "f1") // a file sink takes any path
}
