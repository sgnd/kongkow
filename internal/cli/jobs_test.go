package cli

import (
	"os"
	"path/filepath"
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
	run(t, 0, "post", "status", "--run")
	if j := onlyJob(t); j.Status != store.JobFailed {
		t.Fatalf("status %q", j.Status)
	}
}
