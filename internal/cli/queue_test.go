package cli

import (
	"testing"
	"time"

	"github.com/sgnd/kongkow/internal/out"
	"github.com/sgnd/kongkow/internal/store"
)

func TestParseSlots(t *testing.T) {
	got, err := parseSlots(" 13:00, 9:05,13:00,")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "09:05" || got[1] != "13:00" {
		t.Fatalf("slots %v", got)
	}
	if _, err := parseSlots("25:00"); err == nil {
		t.Fatal("25:00 should be refused")
	}
	if got, _ := parseSlots(""); len(got) != 0 {
		t.Fatalf("empty: %v", got)
	}
}

func TestNextSlot(t *testing.T) {
	d := &store.Dest{Name: "f1", Slots: []string{"09:00", "18:00"}}
	s := &store.State{Jobs: map[string]*store.Job{}}
	now := time.Date(2030, 1, 10, 12, 0, 0, 0, time.Local)
	at, err := nextSlot(s, d, now)
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2030, 1, 10, 18, 0, 0, 0, time.Local); !at.Equal(want) {
		t.Fatalf("got %v want %v", at, want)
	}
	// Taken by a pending job: the next one is tomorrow 09:00.
	s.Jobs["j"] = &store.Job{Dest: "f1", At: at.UTC(), Status: store.JobPending}
	at, _ = nextSlot(s, d, now)
	if want := time.Date(2030, 1, 11, 9, 0, 0, 0, time.Local); !at.Equal(want) {
		t.Fatalf("got %v want %v", at, want)
	}
	// A done job frees nothing new, a cancelled one gives its slot back.
	s.Jobs["j"].Status = store.JobCancelled
	at, _ = nextSlot(s, d, now)
	if want := time.Date(2030, 1, 10, 18, 0, 0, 0, time.Local); !at.Equal(want) {
		t.Fatalf("got %v want %v", at, want)
	}
}

func TestQueueCommand(t *testing.T) {
	setupJobs(t)
	run(t, out.ExitUsage, "post", "queue", "n1", "--to", "f1") // no slots yet
	run(t, 0, "dest", "edit", "f1", "--slots", "09:00,18:00")
	run(t, 0, "post", "queue", "n1", "--to", "f1")
	run(t, 0, "post", "queue", "n1", "--to", "f1")
	st, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	seen := map[time.Time]bool{}
	for _, j := range st.Jobs {
		if seen[j.At] {
			t.Fatalf("two jobs share slot %v", j.At)
		}
		seen[j.At] = true
	}
	if len(seen) != 2 {
		t.Fatalf("want 2 jobs, got %d", len(seen))
	}
	run(t, out.ExitUsage, "dest", "edit", "f1", "--slots", "nine")
	run(t, 0, "dest", "edit", "f1", "--slots", "")
	run(t, out.ExitUsage, "post", "queue", "n1", "--to", "f1")
}
